package api

import (
	"archive/zip"
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/utils"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/mongo"
)

type AlbumMeta struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	CoupleNames []string `json:"couple_names,omitempty"`
}

type AlbumHero struct {
	Title        string `json:"title"`
	Quote        string `json:"quote"`
	CoverMediaID string `json:"cover_media_id,omitempty"`
}

type AlbumMediaURLs struct {
	Thumbnail string `json:"thumbnail,omitempty"`
	Medium    string `json:"medium,omitempty"`
	Large     string `json:"large,omitempty"`
	Original  string `json:"original,omitempty"`
}

type AlbumMediaScores struct {
	Quality  *float64 `json:"quality,omitempty"`
	Emotion  *float64 `json:"emotion,omitempty"`
	Featured *int     `json:"featured,omitempty"`
}

type AlbumMediaAI struct {
	Category  string   `json:"category,omitempty"`
	SceneTags []string `json:"scene_tags,omitempty"`
}

type AlbumMedia struct {
	ID            string           `json:"id"`
	URLs          AlbumMediaURLs   `json:"urls"`
	Width         *int             `json:"width,omitempty"`
	Height        *int             `json:"height,omitempty"`
	AspectRatio   *float64         `json:"aspect_ratio,omitempty"`
	DominantColor string           `json:"dominant_color,omitempty"`
	Orientation   *string          `json:"orientation,omitempty"`
	MimeType      string           `json:"mime_type,omitempty"`
	Scores        AlbumMediaScores `json:"scores,omitempty"`
	AI            AlbumMediaAI     `json:"ai,omitempty"`
	Placeholder   string           `json:"placeholder_url,omitempty"`
}

type AlbumSectionItem struct {
	MediaID      string `json:"media_id"`
	Role         string `json:"role,omitempty"`
	Presentation string `json:"presentation,omitempty"`
	Size         string `json:"size,omitempty"`
}

type AlbumSection struct {
	ID            string             `json:"id"`
	Type          string             `json:"type"`
	Title         string             `json:"title"`
	Quote         string             `json:"quote,omitempty"`
	Layout        string             `json:"layout"`
	Pace          string             `json:"pace,omitempty"`
	Transition    string             `json:"transition,omitempty"`
	DominantColor string             `json:"dominant_color,omitempty"`
	Items         []AlbumSectionItem `json:"items,omitempty"`
	NextCursor    string             `json:"next_cursor,omitempty"`
}

type AlbumStats struct {
	TotalUploads   int            `json:"total_uploads"`
	CategoryCounts map[string]int `json:"category_counts"`
	FeaturedCount  int            `json:"featured_count"`
}

type AlbumResponse struct {
	Album             AlbumMeta             `json:"album"`
	Hero              AlbumHero             `json:"hero"`
	Sections          []AlbumSection        `json:"sections"`
	FeaturedMemoryIDs []string              `json:"featured_memory_ids"`
	Media             map[string]AlbumMedia `json:"media"`
	Stats             AlbumStats            `json:"stats"`
}

func Album(db *mongo.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Query("raw") == "1" {
			wedding, err := getRequestedWedding(c, db)
			if err != nil {
				return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load wedding")
			}
			uploads, err := dao.FindApprovedUploadsByWedding(c.UserContext(), db, wedding.ID)
			if err != nil {
				return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load album")
			}
			album := map[string][]*dto.Upload{}
			for _, u := range uploads {
				album[string(u.Category)] = append(album[string(u.Category)], u)
			}
			return utils.SendJSON(c, fiber.StatusOK, album)
		}

		wedding, err := getRequestedWedding(c, db)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load wedding")
		}

		uploads, err := dao.FindApprovedUploadsByWedding(c.UserContext(), db, c.Params("id"))
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load album")
		}

		response := buildAlbumResponse(wedding, uploads)
		return utils.SendJSON(c, fiber.StatusOK, response)
	}
}

func buildAlbumResponse(wedding *dto.Wedding, uploads []*dto.Upload) AlbumResponse {
	sorted := make([]*dto.Upload, len(uploads))
	copy(sorted, uploads)
	sort.SliceStable(sorted, func(i, j int) bool {
		return getFeaturedScore(sorted[i]) > getFeaturedScore(sorted[j])
	})

	media := make(map[string]AlbumMedia, len(sorted))
	for _, upload := range sorted {
		media[upload.ID] = toAlbumMedia(upload)
	}

	hero := AlbumHero{
		Title:        joinCoupleNames([]string(wedding.CoupleNames)),
		Quote:        "Some memories deserve forever.",
		CoverMediaID: "",
	}
	if len(sorted) > 0 {
		hero.CoverMediaID = sorted[0].ID
	}

	sections := make([]AlbumSection, 0, 5)
	categoryOrder := []dto.UploadCategory{dto.CategoryCeremony, dto.CategoryFamily, dto.CategoryDancing, dto.CategoryCandid, dto.CategoryOther}
	for _, category := range categoryOrder {
		items, nextCursor := makeSectionItems(sorted, category, 8)
		if len(items) == 0 {
			continue
		}
		sections = append(sections, AlbumSection{
			ID:            fmt.Sprintf("section_%s", category),
			Type:          string(category),
			Title:         sectionTitle(category),
			Quote:         sectionQuote(category),
			Layout:        sectionLayout(category),
			Pace:          sectionPace(category),
			Transition:    sectionTransition(category),
			DominantColor: sectionDominantColor(category),
			Items:         items,
			NextCursor:    nextCursor,
		})
	}

	featuredIDs := topItemIDs(sorted, 6)

	stats := AlbumStats{
		TotalUploads:   len(uploads),
		CategoryCounts: categorizeCounts(uploads),
		FeaturedCount:  len(featuredIDs),
	}

	return AlbumResponse{
		Album: AlbumMeta{
			ID:          wedding.ID,
			Title:       joinCoupleNames([]string(wedding.CoupleNames)),
			CoupleNames: []string(wedding.CoupleNames),
		},
		Hero:              hero,
		Sections:          sections,
		FeaturedMemoryIDs: featuredIDs,
		Media:             media,
		Stats:             stats,
	}
}

func makeSectionItems(uploads []*dto.Upload, category dto.UploadCategory, limit int) ([]AlbumSectionItem, string) {
	items := make([]AlbumSectionItem, 0, limit)
	total := 0
	for _, u := range uploads {
		if u.Analysis.Category != category {
			continue
		}
		total++
		if len(items) < limit {
			items = append(items, toAlbumSectionItem(u))
		}
	}
	nextCursor := ""
	if total > limit && len(items) > 0 {
		nextCursor = items[len(items)-1].MediaID
	}
	return items, nextCursor
}

func topItemIDs(uploads []*dto.Upload, limit int) []string {
	itemIDs := make([]string, 0, min(limit, len(uploads)))
	for i, u := range uploads {
		if i >= limit {
			break
		}
		itemIDs = append(itemIDs, u.ID)
	}
	return itemIDs
}

func toAlbumSectionItem(upload *dto.Upload) AlbumSectionItem {
	role := itemRole(upload)
	return AlbumSectionItem{
		MediaID:      upload.ID,
		Role:         role,
		Presentation: presentationForRole(role),
		Size:         sizeForRole(role),
	}
}

func toAlbumMedia(upload *dto.Upload) AlbumMedia {
	return AlbumMedia{
		ID: upload.ID,
		URLs: AlbumMediaURLs{
			Thumbnail: assetURL(upload.Storage.ThumbnailURL, upload.Storage.MediumURL, upload.Storage.OriginalURL),
			Medium:    assetURL(upload.Storage.MediumURL, upload.Storage.OriginalURL),
			Large:     assetURL(upload.Storage.MediumURL, upload.Storage.OriginalURL),
			Original:  upload.Storage.OriginalURL,
		},
		Width:       upload.Metadata.Width,
		Height:      upload.Metadata.Height,
		AspectRatio: upload.Metadata.AspectRatio,
		Orientation: upload.Orientation,
		MimeType:    upload.MimeType,
		Scores: AlbumMediaScores{
			Quality:  upload.Analysis.QualityScore,
			Emotion:  upload.Analysis.EmotionScore,
			Featured: upload.Analysis.FeaturedScore,
		},
		AI: AlbumMediaAI{
			Category:  string(upload.Analysis.Category),
			SceneTags: upload.Analysis.SceneTags,
		},
		Placeholder: assetURL(upload.Storage.ThumbnailURL, upload.Storage.MediumURL, upload.Storage.OriginalURL),
	}
}

func itemRole(upload *dto.Upload) string {
	score := getFeaturedScore(upload)
	switch {
	case score >= 95:
		return "hero"
	case score >= 85:
		return "banner"
	case score >= 70:
		return "supporting"
	default:
		return "filler"
	}
}

func presentationForRole(role string) string {
	switch role {
	case "hero":
		return "feature"
	case "banner":
		return "highlight"
	case "supporting":
		return "support"
	default:
		return "thumbnail"
	}
}

func sizeForRole(role string) string {
	switch role {
	case "hero":
		return "xl"
	case "banner":
		return "lg"
	case "supporting":
		return "md"
	default:
		return "sm"
	}
}

func assetURL(urls ...string) string {
	for _, u := range urls {
		if u != "" {
			return u
		}
	}
	return ""
}

func joinCoupleNames(names []string) string {
	if len(names) == 0 {
		return ""
	}
	return strings.Join(names, " & ")
}

func splitCoupleNames(names string) []string {
	if names == "" {
		return nil
	}
	parts := strings.Split(names, " & ")
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

func sectionPace(category dto.UploadCategory) string {
	switch category {
	case dto.CategoryCeremony:
		return "slow"
	case dto.CategoryFamily:
		return "calm"
	case dto.CategoryDancing:
		return "energetic"
	case dto.CategoryCandid:
		return "upbeat"
	default:
		return "steady"
	}
}

func sectionTransition(category dto.UploadCategory) string {
	switch category {
	case dto.CategoryCeremony:
		return "cinematic_fade"
	case dto.CategoryFamily:
		return "soft_pan"
	case dto.CategoryDancing:
		return "film_slide"
	case dto.CategoryCandid:
		return "quick_stitch"
	default:
		return "fade"
	}
}

func sectionDominantColor(category dto.UploadCategory) string {
	switch category {
	case dto.CategoryCeremony:
		return "#E7D8C9"
	case dto.CategoryFamily:
		return "#D8E7C9"
	case dto.CategoryDancing:
		return "#F6D0E6"
	case dto.CategoryCandid:
		return "#C9D8E7"
	default:
		return "#E7E1C9"
	}
}

func getFeaturedScore(upload *dto.Upload) float64 {
	if upload.Analysis.FeaturedScore != nil {
		return float64(*upload.Analysis.FeaturedScore)
	}
	base := 0.0
	if upload.Analysis.QualityScore != nil {
		base += *upload.Analysis.QualityScore * 60.0
	}
	if upload.Analysis.EmotionScore != nil {
		base += *upload.Analysis.EmotionScore * 30.0
	}
	if upload.Analysis.DetectedFaces != nil {
		base += float64(min(*upload.Analysis.DetectedFaces, 10)) * 1.5
	}
	switch upload.Analysis.Category {
	case dto.CategoryCeremony, dto.CategoryFamily:
		base += 10.0
	}
	if base > 100 {
		base = 100
	}
	return base
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func filterTopByCategory(uploads []*dto.Upload, category dto.UploadCategory, limit int) []*dto.Upload {
	filtered := make([]*dto.Upload, 0, limit)
	for _, u := range uploads {
		if u.Analysis.Category == category {
			filtered = append(filtered, u)
			if len(filtered) >= limit {
				break
			}
		}
	}
	return filtered
}

func topN(uploads []*dto.Upload, limit int) []*dto.Upload {
	if len(uploads) <= limit {
		return uploads
	}
	return uploads[:limit]
}

func categorizeCounts(uploads []*dto.Upload) map[string]int {
	counts := make(map[string]int)
	for _, u := range uploads {
		counts[string(u.Analysis.Category)]++
	}
	return counts
}

func sectionLayout(category dto.UploadCategory) string {
	switch category {
	case dto.CategoryCeremony:
		return "cinematic"
	case dto.CategoryFamily:
		return "masonry"
	case dto.CategoryDancing:
		return "filmstrip"
	case dto.CategoryCandid:
		return "collage"
	default:
		return "masonry"
	}
}

func sectionTitle(category dto.UploadCategory) string {
	switch category {
	case dto.CategoryCeremony:
		return "The Ceremony"
	case dto.CategoryFamily:
		return "Family & Portraits"
	case dto.CategoryDancing:
		return "Dance Floor"
	case dto.CategoryCandid:
		return "Candid Moments"
	default:
		return "More Memories"
	}
}

func sectionQuote(category dto.UploadCategory) string {
	switch category {
	case dto.CategoryCeremony:
		return "The vows, the first look, the magic."
	case dto.CategoryFamily:
		return "Loved ones who were there every step of the way."
	case dto.CategoryDancing:
		return "The energy that filled the room after dark."
	case dto.CategoryCandid:
		return "Unscripted moments that feel closest to the heart."
	default:
		return "A collection of the night’s best memories."
	}
}

func Highlights(db *mongo.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		wedding, err := getRequestedWedding(c, db)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load wedding")
		}
		highlights, err := dao.FindRandomPhotoHighlights(c.UserContext(), db, wedding.ID, 20)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load highlights")
		}
		return utils.SendJSON(c, fiber.StatusOK, highlights)
	}
}

func getRequestedWedding(c *fiber.Ctx, db *mongo.Database) (*dto.Wedding, error) {
	id := c.Params("id")
	if id == "" {
		id = c.Params("slug")
	}
	if id == "" {
		return nil, errors.New("missing wedding identifier")
	}

	wedding, err := dao.FindWeddingByID(c.UserContext(), db, id)
	if err == nil {
		return wedding, nil
	}
	if !errors.Is(err, dao.ErrNoRows) {
		return nil, err
	}
	return dao.FindWeddingBySlug(c.UserContext(), db, id)
}

func Download(db *mongo.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		weddingID := c.Params("id")

		wedding, err := dao.FindWeddingByID(c.UserContext(), db, weddingID)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load wedding")
		}
		if wedding.Tier == dto.TierElopement {
			return utils.SendErrorResponse(c, fiber.StatusPaymentRequired, "bulk download requires Heritage or Legacy tier")
		}

		uploads, err := dao.FindApprovedUploadsByWedding(c.UserContext(), db, weddingID)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load uploads")
		}

		c.Set("Content-Type", "application/zip")
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s-album.zip\"", weddingID))

		c.Context().SetBodyStreamWriter(fasthttp.StreamWriter(func(w *bufio.Writer) {
			zw := zip.NewWriter(w)
			defer zw.Close()

			client := &http.Client{Timeout: 30 * time.Second}
			for _, upload := range uploads {
				resp, err := client.Get(upload.FileURL)
				if err != nil {
					continue
				}
				parsed, _ := url.Parse(upload.FileURL)
				fileName := fmt.Sprintf("%s%s", upload.ID, parsedExtension(parsed.Path))
				f, err := zw.Create(fileName)
				if err != nil {
					resp.Body.Close()
					continue
				}
				_, _ = io.Copy(f, resp.Body)
				resp.Body.Close()
			}
		}))

		return nil
	}
}

func parsedExtension(path string) string {
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '.' {
			return path[i:]
		}
		if path[i] == '/' {
			break
		}
	}
	return ""
}
