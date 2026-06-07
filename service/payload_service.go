package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/mongo"
)

const payloadVersion = "1.0"

type PayloadService struct {
	db *mongo.Database
}

func NewPayloadService(db *mongo.Database) *PayloadService {
	return &PayloadService{db: db}
}

func (p *PayloadService) DB() *mongo.Database { return p.db }

func (p *PayloadService) BuildPayload(ctx context.Context, weddingID string) (*dto.AlbumPayload, error) {
	albumCfg, err := dao.FindAlbumConfig(ctx, p.db, weddingID)
	if err != nil {
		return nil, fmt.Errorf("album config not found: %w", err)
	}

	acts, err := dao.FindStoryActs(ctx, p.db, weddingID)
	if err != nil {
		return nil, fmt.Errorf("story acts not found: %w", err)
	}

	wedding, err := dao.FindWeddingByID(ctx, p.db, weddingID)
	if err != nil {
		return nil, fmt.Errorf("wedding not found: %w", err)
	}

	// Bulk-fetch all uploads so blocks get full asset objects (URL + dims).
	uploads, _ := dao.FindApprovedUploadsByWedding(ctx, p.db, weddingID)
	uploadMap := make(map[string]*dto.Upload, len(uploads))
	for _, u := range uploads {
		uploadMap[u.ID] = u
	}

	style := albumCfg.Style
	if style == "" {
		style = dto.AlbumStyleCinematic
	}

	assetFor := func(id string) *dto.AssetObject {
		if id == "" {
			return nil
		}
		obj := &dto.AssetObject{ID: id}
		if u, ok := uploadMap[id]; ok {
			obj.URL = u.Storage.OriginalURL
			if u.Metadata.Width != nil {
				obj.Width = *u.Metadata.Width
			}
			if u.Metadata.Height != nil {
				obj.Height = *u.Metadata.Height
			}
			if u.Metadata.AspectRatio != nil {
				obj.AspectRatio = *u.Metadata.AspectRatio
			}
		}
		return obj
	}

	assetsFor := func(ids []string) []dto.AssetObject {
		out := make([]dto.AssetObject, 0, len(ids))
		for _, id := range ids {
			if a := assetFor(id); a != nil {
				out = append(out, *a)
			}
		}
		return out
	}

	filter := filterForStyle(style, wedding.WeddingMood)

	coupleTitle := coupleNamesTitle(wedding.CoupleNames)
	dateSubtitle := wedding.WeddingDate.Format("2 January 2006")
	if wedding.Venue != "" {
		dateSubtitle += " · " + wedding.Venue
	}

	payload := &dto.AlbumPayload{
		Version:  payloadVersion,
		AlbumID:  weddingID,
		Style:    style,
		Theme:    themeForStyle(style, wedding.WeddingMood, wedding.WeddingTheme),
		Chapters: []dto.AlbumChapter{},
	}

	// Cover from the first act's hero.
	if len(acts) > 0 && acts[0].HeroPhotoID != "" {
		payload.Cover = dto.AlbumCoverBlock{
			Image:    assetFor(acts[0].HeroPhotoID),
			Title:    coupleTitle,
			Subtitle: dateSubtitle,
		}
	}

	// Optional ambient soundtrack per style.
	payload.Soundtrack = soundtrackForStyle(style)

	for _, act := range acts {
		chapter := dto.AlbumChapter{
			ID:    uuid.NewString(),
			Title: act.Title,
			Act:   act.Label,
		}

		blocks := []dto.AlbumBlock{}

		// Section title block.
		blocks = append(blocks, dto.AlbumBlock{
			ID:     uuid.NewString(),
			Type:   "section_title",
			Title:  act.Title,
			Motion: motionFor("section_title", style),
		})

		// Quote block.
		if act.Quote != "" {
			blocks = append(blocks, dto.AlbumBlock{
				ID:     uuid.NewString(),
				Type:   "quote",
				Text:   act.Quote,
				Layout: "centered",
				Motion: motionFor("quote", style),
			})
		}

		if len(act.PhotoIDs) == 0 {
			chapter.Blocks = blocks
			payload.Chapters = append(payload.Chapters, chapter)
			continue
		}

		// Hero fullscreen block.
		if act.HeroPhotoID != "" {
			heroAsset := assetFor(act.HeroPhotoID)
			blocks = append(blocks, dto.AlbumBlock{
				ID:     uuid.NewString(),
				Type:   "fullscreen_image",
				Image:  heroAsset,
				Height: "full",
				Overlay: &dto.BlockOverlay{
					Title:       act.Title,
					Alignment:   "bottom-left",
					RevealDelay: 0.6,
				},
				Motion: motionFor("fullscreen_image", style),
				Filter: filter,
			})
		}

		// Remaining photos.
		remaining := idsWithout(act.PhotoIDs, act.HeroPhotoID)
		if len(remaining) > 0 {
			switch {
			case len(remaining) <= 3:
				blocks = append(blocks, dto.AlbumBlock{
					ID:      uuid.NewString(),
					Type:    "masonry",
					Images:  assetsFor(remaining),
					Columns: len(remaining),
					Motion:  motionFor("masonry", style),
					Filter:  filter,
				})
			case len(remaining) <= 6:
				blocks = append(blocks, dto.AlbumBlock{
					ID:     uuid.NewString(),
					Type:   "filmstrip",
					Images: assetsFor(remaining),
					Motion: motionFor("filmstrip", style),
					Filter: filter,
				})
			default:
				blocks = append(blocks, dto.AlbumBlock{
					ID:      uuid.NewString(),
					Type:    "gallery_grid",
					Images:  assetsFor(remaining[:6]),
					Columns: 3,
					Motion:  motionFor("gallery_grid", style),
					Filter:  filter,
				})
				if len(remaining) > 6 {
					blocks = append(blocks, dto.AlbumBlock{
						ID:     uuid.NewString(),
						Type:   "filmstrip",
						Images: assetsFor(remaining[6:]),
						Motion: motionFor("filmstrip", style),
						Filter: filter,
					})
				}
			}
		}

		chapter.Blocks = blocks
		payload.Chapters = append(payload.Chapters, chapter)
	}

	// Finale chapter — no act, just the closing block.
	if len(acts) > 0 {
		last := acts[len(acts)-1]
		heroID := last.HeroPhotoID
		if heroID == "" && len(last.PhotoIDs) > 0 {
			heroID = last.PhotoIDs[len(last.PhotoIDs)-1]
		}
		if heroID != "" {
			payload.Chapters = append(payload.Chapters, dto.AlbumChapter{
				ID:    uuid.NewString(),
				Title: "Finale",
				Act:   dto.ActFinalDance,
				Blocks: []dto.AlbumBlock{
					{
						ID:              uuid.NewString(),
						Type:            "finale",
						BackgroundImage: assetFor(heroID),
						Title:           "The End — and the Beginning",
						Motion:          motionFor("finale", style),
					},
				},
			})
		}
	}

	return payload, nil
}

// --- style helpers ---

func themeForStyle(style dto.AlbumStyle, mood, theme string) dto.AlbumTheme {
	switch style {
	case dto.AlbumStyleRomantic:
		return dto.AlbumTheme{
			PrimaryColor:   "#f5e6e8",
			SecondaryColor: "#c2847a",
			Font:           "Cormorant Garamond",
			Mood:           moodLabel(mood, "soft"),
		}
	default:
		return dto.AlbumTheme{
			PrimaryColor:   "#1a1a1a",
			SecondaryColor: "#c9a96e",
			Font:           "Playfair Display",
			Mood:           moodLabel(mood, "dramatic"),
		}
	}
}

func filterForStyle(style dto.AlbumStyle, mood string) *dto.BlockFilter {
	// Adjust vignette and grade based on wedding mood when available.
	switch mood {
	case "bright", "joyful", "fun":
		return &dto.BlockFilter{Grade: "warm", Vignette: 0.05}
	case "moody", "dark", "dramatic":
		return &dto.BlockFilter{Grade: "muted", Vignette: 0.35}
	}
	switch style {
	case dto.AlbumStyleRomantic:
		return &dto.BlockFilter{Grade: "warm", Vignette: 0.10}
	default:
		return &dto.BlockFilter{Grade: "muted", Vignette: 0.25}
	}
}

func moodLabel(weddingMood, fallback string) string {
	if weddingMood != "" {
		return weddingMood
	}
	return fallback
}

func coupleNamesTitle(names []string) string {
	switch len(names) {
	case 0:
		return "Our Wedding Story"
	case 1:
		return names[0]
	default:
		return names[0] + " & " + names[1]
	}
}

func soundtrackForStyle(style dto.AlbumStyle) *dto.AlbumSoundtrack {
	// URL is intentionally empty — the couple can attach one via the dashboard.
	return &dto.AlbumSoundtrack{
		Autoplay: false,
		Volume:   0.35,
	}
}

// motionFor returns pacing hints per block type.
// Cinematic style uses slower, more dramatic reveals; romantic uses softer fades.
func motionFor(blockType string, style dto.AlbumStyle) *dto.BlockMotion {
	slow := style == dto.AlbumStyleCinematic

	switch blockType {
	case "fullscreen_image":
		d := 1.0
		if slow {
			d = 1.4
		}
		return &dto.BlockMotion{Reveal: "fade-in", KenBurns: true, Parallax: 0.3, Duration: d}
	case "hero_image":
		d := 0.7
		if slow {
			d = 1.0
		}
		return &dto.BlockMotion{Reveal: "fade-up", Duration: d}
	case "masonry", "gallery_grid":
		return &dto.BlockMotion{Reveal: "fade-up", Duration: 0.6}
	case "filmstrip":
		return &dto.BlockMotion{Reveal: "slide-left", Duration: 0.5}
	case "split_story":
		return &dto.BlockMotion{Reveal: "fade-up", Duration: 0.9}
	case "quote", "section_title":
		return &dto.BlockMotion{Reveal: "fade-up", Duration: 0.5}
	case "finale":
		d := 1.8
		if slow {
			d = 2.4
		}
		return &dto.BlockMotion{Reveal: "fade-in", KenBurns: true, Parallax: 0.2, Duration: d}
	}
	return &dto.BlockMotion{Reveal: "fade-up", Duration: 0.6}
}

func idsWithout(ids []string, exclude string) []string {
	if exclude == "" {
		return ids
	}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		if id != exclude {
			out = append(out, id)
		}
	}
	return out
}
