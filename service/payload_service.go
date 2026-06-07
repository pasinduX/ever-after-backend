package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	"go.mongodb.org/mongo-driver/mongo"
)

type PayloadService struct {
	db *mongo.Database
}

func NewPayloadService(db *mongo.Database) *PayloadService {
	return &PayloadService{db: db}
}

func (p *PayloadService) BuildPayload(ctx context.Context, weddingID string) (*dto.AlbumPayload, error) {
	albumCfg, err := dao.FindAlbumConfig(ctx, p.db, weddingID)
	if err != nil {
		return nil, fmt.Errorf("album config not found: %w", err)
	}

	acts, err := dao.FindStoryActs(ctx, p.db, weddingID)
	if err != nil {
		return nil, fmt.Errorf("story acts not found: %w", err)
	}

	style := albumCfg.Style
	if style == "" {
		style = dto.AlbumStyleCinematic
	}

	theme := themeForStyle(style)

	payload := &dto.AlbumPayload{
		AlbumID: weddingID,
		Style:   style,
		Theme:   theme,
		Blocks:  []dto.AlbumBlock{},
	}

	// Cover from the first act's hero
	if len(acts) > 0 && acts[0].HeroPhotoID != "" {
		payload.Cover = dto.AlbumCoverBlock{
			ImageID:  acts[0].HeroPhotoID,
			Title:    "Our Wedding Story",
			Subtitle: acts[0].Title,
		}
	}

	for _, act := range acts {
		// Section title block
		payload.Blocks = append(payload.Blocks, dto.AlbumBlock{
			ID:    uuid.NewString(),
			Type:  "section_title",
			Title: act.Title,
		})

		// Quote block if the act has one
		if act.Quote != "" {
			payload.Blocks = append(payload.Blocks, dto.AlbumBlock{
				ID:   uuid.NewString(),
				Type: "quote",
				Text: act.Quote,
			})
		}

		if len(act.PhotoIDs) == 0 {
			continue
		}

		// Hero photo fullscreen
		if act.HeroPhotoID != "" {
			payload.Blocks = append(payload.Blocks, dto.AlbumBlock{
				ID:      uuid.NewString(),
				Type:    "fullscreen_image",
				ImageID: act.HeroPhotoID,
				Height:  "full",
				Overlay: &dto.BlockOverlay{
					Title:     act.Title,
					Alignment: "bottom-left",
				},
			})
		}

		// Remaining photos go into a masonry/filmstrip based on count
		remaining := photoIDsWithout(act.PhotoIDs, act.HeroPhotoID)
		if len(remaining) == 0 {
			continue
		}

		switch {
		case len(remaining) <= 3:
			payload.Blocks = append(payload.Blocks, dto.AlbumBlock{
				ID:      uuid.NewString(),
				Type:    "masonry",
				Images:  remaining,
				Columns: len(remaining),
			})
		case len(remaining) <= 6:
			payload.Blocks = append(payload.Blocks, dto.AlbumBlock{
				ID:     uuid.NewString(),
				Type:   "filmstrip",
				Images: remaining,
			})
		default:
			// Large acts get a gallery grid + filmstrip overflow
			payload.Blocks = append(payload.Blocks, dto.AlbumBlock{
				ID:      uuid.NewString(),
				Type:    "gallery_grid",
				Images:  remaining[:6],
				Columns: 3,
			})
			if len(remaining) > 6 {
				payload.Blocks = append(payload.Blocks, dto.AlbumBlock{
					ID:     uuid.NewString(),
					Type:   "filmstrip",
					Images: remaining[6:],
				})
			}
		}
	}

	// Finale block using last act hero
	if len(acts) > 0 {
		last := acts[len(acts)-1]
		heroID := last.HeroPhotoID
		if heroID == "" && len(last.PhotoIDs) > 0 {
			heroID = last.PhotoIDs[len(last.PhotoIDs)-1]
		}
		if heroID != "" {
			payload.Blocks = append(payload.Blocks, dto.AlbumBlock{
				ID:              uuid.NewString(),
				Type:            "finale",
				BackgroundImage: heroID,
				Title:           "The End — and the Beginning",
			})
		}
	}

	return payload, nil
}

func themeForStyle(style dto.AlbumStyle) dto.AlbumTheme {
	switch style {
	case dto.AlbumStyleRomantic:
		return dto.AlbumTheme{
			PrimaryColor:   "#f5e6e8",
			SecondaryColor: "#c2847a",
			Font:           "Cormorant Garamond",
			Mood:           "soft",
		}
	default: // cinematic
		return dto.AlbumTheme{
			PrimaryColor:   "#1a1a1a",
			SecondaryColor: "#c9a96e",
			Font:           "Playfair Display",
			Mood:           "dramatic",
		}
	}
}

func photoIDsWithout(ids []string, exclude string) []string {
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
