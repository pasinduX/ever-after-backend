package api

import (
	"archive/zip"
	"bufio"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/storyvows/backend/dao"
	"github.com/storyvows/backend/dto"
	"github.com/storyvows/backend/utils"
	"github.com/valyala/fasthttp"
	"go.mongodb.org/mongo-driver/mongo"
)

func Album(db *mongo.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		uploads, err := dao.FindApprovedUploadsByWedding(c.UserContext(), db, c.Params("id"))
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load album")
		}
		album := map[string][]*dto.Upload{}
		for _, u := range uploads {
			album[string(u.Category)] = append(album[string(u.Category)], u)
		}
		return utils.SendJSON(c, fiber.StatusOK, album)
	}
}

func Highlights(db *mongo.Database) fiber.Handler {
	return func(c *fiber.Ctx) error {
		highlights, err := dao.FindRandomPhotoHighlights(c.UserContext(), db, c.Params("id"), 20)
		if err != nil {
			return utils.SendErrorResponse(c, fiber.StatusInternalServerError, "failed to load highlights")
		}
		return utils.SendJSON(c, fiber.StatusOK, highlights)
	}
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
