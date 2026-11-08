package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
)

const (
	// pageRequestTimeout allows for slow reasoning models writing a long
	// document; pageStageBudget bounds the whole compose-and-repair loop.
	pageRequestTimeout = 6 * time.Minute
	pageStageBudget    = 18 * time.Minute
)

// errQuotaExhausted means the OpenAI account has no credits left. Retrying or
// repairing cannot help, and the app tells the couple why.
var errQuotaExhausted = errors.New("the OpenAI account has no credits left")

type pageComposition struct {
	Palette    []dto.PageColor `json:"palette"`
	Typography struct {
		Display string `json:"display"`
		Body    string `json:"body"`
	} `json:"typography"`
	DesignRationale string `json:"design_rationale"`
	HTML            string `json:"html"`
}

// pageRepair carries a rejected document back to the model with the exact
// problems, so it fixes them instead of starting over.
type pageRepair struct {
	PreviousHTML string   `json:"previous_html"`
	Issues       []string `json:"issues"`
}

type pageUsage struct {
	Model        string
	InputTokens  int
	OutputTokens int
	Latency      time.Duration
}

func pageCompositionSchema() map[string]any {
	str := map[string]any{"type": "string"}
	return map[string]any{
		"type": "object", "additionalProperties": false,
		"required": []string{"palette", "typography", "design_rationale", "html"},
		"properties": map[string]any{
			"palette": map[string]any{"type": "array", "items": map[string]any{
				"type": "object", "additionalProperties": false,
				"required":   []string{"role", "hex"},
				"properties": map[string]any{"role": str, "hex": str},
			}},
			"typography": map[string]any{
				"type": "object", "additionalProperties": false,
				"required":   []string{"display", "body"},
				"properties": map[string]any{"display": str, "body": str},
			},
			"design_rationale": str,
			"html":             str,
		},
	}
}

// GenerateAlbumPage has the art director design a new album page version for
// a wedding from its current acts and photos. It never fails silently: every
// outcome, including a fallback and its reason, is stored as a version.
func (s *AlbumService) GenerateAlbumPage(ctx context.Context, weddingID string) (*dto.AlbumPage, error) {
	page, err := dao.StartAlbumPage(ctx, s.db, uuid.NewString(), weddingID)
	if err != nil {
		return nil, err
	}
	finish := func(reason string, diagnostics []string) (*dto.AlbumPage, error) {
		page.Status = dto.AlbumPageFallback
		page.FallbackReason = reason
		page.Diagnostics = diagnostics
		// Store the outcome even if the job's own deadline has passed.
		saveCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		return page, dao.SaveAlbumPage(saveCtx, s.db, page)
	}

	wedding, err := dao.FindWeddingByID(ctx, s.db, weddingID)
	if err != nil {
		return finish(dto.AlbumPageUnavailable, []string{"load wedding: " + err.Error()})
	}
	acts, err := dao.FindStoryActs(ctx, s.db, weddingID)
	if err != nil && !errors.Is(err, dao.ErrNoRows) {
		return finish(dto.AlbumPageUnavailable, []string{"load acts: " + err.Error()})
	}
	uploads, err := dao.FindApprovedUploadsByWedding(ctx, s.db, weddingID)
	if err != nil {
		return finish(dto.AlbumPageUnavailable, []string{"load photos: " + err.Error()})
	}

	brief, rules := buildPageBrief(wedding, acts, uploads)
	if len(brief.Photos) == 0 {
		return finish(dto.AlbumPageNoPhotos, nil)
	}
	if s.cfg.OpenAIAPIKey == "" || !s.cfg.AlbumPageEnabled {
		return finish(dto.AlbumPageNotConfigured, nil)
	}

	ctx, cancel := context.WithTimeout(ctx, pageStageBudget)
	defer cancel()

	designPage(ctx, s.composePage, brief, rules, s.cfg.AlbumPageRepairs, page)
	if page.Status == dto.AlbumPageReady {
		slog.Info("album page ready", "wedding_id", weddingID, "version", page.Version,
			"attempts", page.Attempts, "model", page.Model, "output_tokens", page.OutputTokens,
			"seconds", page.LatencyMS/1000)
		if err := dao.SaveAlbumPage(context.Background(), s.db, page); err != nil {
			return page, err
		}
		// A fresh design replaces whatever guests were seeing.
		page.Active = true
		return page, dao.ActivateAlbumPage(context.Background(), s.db, weddingID, page.Version)
	}
	slog.Warn("album page fell back", "wedding_id", weddingID, "version", page.Version,
		"reason", page.FallbackReason, "attempts", page.Attempts, "diagnostics", page.Diagnostics)
	return finish(page.FallbackReason, page.Diagnostics)
}

type composeFunc func(ctx context.Context, brief pageBrief, repair *pageRepair) (pageComposition, pageUsage, error)

// designPage asks the model for a page, then feeds the checker's problems
// back for up to maxRepairs corrections. It fills in page: ready with its
// document, or a fallback reason and diagnostics, plus usage either way.
func designPage(ctx context.Context, compose composeFunc, brief pageBrief, rules albumPageRules, maxRepairs int, page *dto.AlbumPage) {
	var repair *pageRepair
	var usage pageUsage
	page.Status, page.FallbackReason, page.Diagnostics = dto.AlbumPageFallback, dto.AlbumPageInvalid, nil
	for attempt := 0; attempt <= maxRepairs; attempt++ {
		page.Attempts = attempt + 1
		composition, used, err := compose(ctx, brief, repair)
		usage.Model = firstNonEmpty(used.Model, usage.Model)
		usage.InputTokens += used.InputTokens
		usage.OutputTokens += used.OutputTokens
		usage.Latency += used.Latency
		page.Model, page.InputTokens, page.OutputTokens, page.LatencyMS =
			usage.Model, usage.InputTokens, usage.OutputTokens, usage.Latency.Milliseconds()
		if err != nil {
			page.FallbackReason = dto.AlbumPageUnavailable
			if errors.Is(err, errQuotaExhausted) {
				page.FallbackReason = dto.AlbumPageQuotaExhausted
			}
			page.Diagnostics = []string{err.Error()}
			return
		}
		issues := checkAlbumPage(composition.HTML, rules)
		if len(issues) == 0 {
			page.Status = dto.AlbumPageReady
			page.HTML = composition.HTML
			page.Palette = composition.Palette
			page.DisplayFont = composition.Typography.Display
			page.BodyFont = composition.Typography.Body
			page.DesignRationale = composition.DesignRationale
			page.FallbackReason, page.Diagnostics = "", nil
			return
		}
		page.FallbackReason, page.Diagnostics = dto.AlbumPageInvalid, issues
		repair = &pageRepair{PreviousHTML: composition.HTML, Issues: issues}
		if ctx.Err() != nil {
			return
		}
	}
}

// composePage makes one art-director call with a strict JSON schema.
func (s *AlbumService) composePage(ctx context.Context, brief pageBrief, repair *pageRepair) (pageComposition, pageUsage, error) {
	input, err := json.Marshal(struct {
		pageBrief
		Repair *pageRepair `json:"repair,omitempty"`
	}{brief, repair})
	if err != nil {
		return pageComposition{}, pageUsage{}, err
	}
	body, err := json.Marshal(map[string]any{
		"model": s.cfg.OpenAIModelPage,
		"input": []any{
			map[string]any{"role": "system", "content": weddingArtDirectorPrompt},
			map[string]any{"role": "user", "content": "BEGIN_UNTRUSTED_STRUCTURED_INPUT\n" + string(input) + "\nEND_UNTRUSTED_STRUCTURED_INPUT"},
		},
		"text": map[string]any{"format": map[string]any{
			"type": "json_schema", "name": "wedding_album_page", "strict": true,
			"schema": pageCompositionSchema(),
		}},
	})
	if err != nil {
		return pageComposition{}, pageUsage{}, err
	}

	reqCtx, cancel := context.WithTimeout(ctx, pageRequestTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, openAIEndpoint, bytes.NewReader(body))
	if err != nil {
		return pageComposition{}, pageUsage{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.cfg.OpenAIAPIKey)
	req.Header.Set("Content-Type", "application/json")

	started := time.Now()
	resp, err := s.pageClient.Do(req)
	usage := pageUsage{Model: s.cfg.OpenAIModelPage, Latency: time.Since(started)}
	if err != nil {
		return pageComposition{}, usage, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	usage.Latency = time.Since(started)
	if err != nil {
		return pageComposition{}, usage, err
	}
	if resp.StatusCode >= 300 {
		if isQuotaError(resp.StatusCode, data) {
			return pageComposition{}, usage, errQuotaExhausted
		}
		return pageComposition{}, usage, fmt.Errorf("openai response %d: %s", resp.StatusCode, truncateRunes(string(data), 500))
	}

	var meta struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens  int `json:"input_tokens"`
			OutputTokens int `json:"output_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(data, &meta)
	usage.Model = firstNonEmpty(meta.Model, usage.Model)
	usage.InputTokens, usage.OutputTokens = meta.Usage.InputTokens, meta.Usage.OutputTokens

	text, err := responseOutputText(data)
	if err != nil {
		return pageComposition{}, usage, err
	}
	var composition pageComposition
	if err := json.Unmarshal([]byte(text), &composition); err != nil {
		return pageComposition{}, usage, fmt.Errorf("decode album page: %w", err)
	}
	return composition, usage, nil
}

func isQuotaError(status int, body []byte) bool {
	return status == http.StatusTooManyRequests &&
		(bytes.Contains(body, []byte("insufficient_quota")) || bytes.Contains(body, []byte("credit_balance_exhausted")))
}

// AlbumPageState is what the app needs to show a wedding's album page: the
// newest version's progress, and the newest page that is ready to show.
type AlbumPageState struct {
	Latest *dto.AlbumPage `json:"latest"`
	Ready  *dto.AlbumPage `json:"ready"`
}

func (s *AlbumService) PageState(ctx context.Context, weddingID string) (*AlbumPageState, error) {
	state := &AlbumPageState{}
	latest, err := dao.FindLatestAlbumPage(ctx, s.db, weddingID)
	if err != nil && !errors.Is(err, dao.ErrNoRows) {
		return nil, err
	}
	state.Latest = latest
	if latest != nil {
		ready, err := s.shownPage(ctx, weddingID)
		if err != nil {
			return nil, err
		}
		if ready != nil && ready.Version == latest.Version {
			ready = latest
		}
		state.Ready = ready
	}
	if state.Latest != nil && state.Latest != state.Ready {
		// The progress record never needs the document itself.
		summary := *state.Latest
		summary.HTML = ""
		state.Latest = &summary
	}
	return state, nil
}

// shownPage is the version guests see: the activated one, else the newest
// ready one (for pages stored before versions could be activated).
func (s *AlbumService) shownPage(ctx context.Context, weddingID string) (*dto.AlbumPage, error) {
	page, err := dao.FindActiveAlbumPage(ctx, s.db, weddingID)
	if errors.Is(err, dao.ErrNoRows) {
		page, err = dao.FindLatestReadyAlbumPage(ctx, s.db, weddingID)
	}
	if errors.Is(err, dao.ErrNoRows) {
		return nil, nil
	}
	return page, err
}

// ShownPage returns the page guests see, or nil while there is none.
func (s *AlbumService) ShownPage(ctx context.Context, weddingID string) (*dto.AlbumPage, error) {
	return s.shownPage(ctx, weddingID)
}

func (s *AlbumService) PageVersions(ctx context.Context, weddingID string) ([]*dto.AlbumPage, error) {
	return dao.ListAlbumPages(ctx, s.db, weddingID)
}

// ActivatePageVersion restores an earlier ready design for guests.
func (s *AlbumService) ActivatePageVersion(ctx context.Context, weddingID string, version int) error {
	err := dao.ActivateAlbumPage(ctx, s.db, weddingID, version)
	if errors.Is(err, dao.ErrNoRows) {
		return errAlbumPageVersionNotReady
	}
	return err
}

var errAlbumPageVersionNotReady = errors.New("that album design is not available to restore")

// IsPageVersionNotReady reports whether an activation failed because the
// version doesn't exist or never became ready.
func IsPageVersionNotReady(err error) bool { return errors.Is(err, errAlbumPageVersionNotReady) }
