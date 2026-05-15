package functions

import (
	"bytes"
	"encoding/base64"

	"github.com/skip2/go-qrcode"
)

// GenerateQRDataURL generates a base64-encoded PNG data URL for the given content string.
func GenerateQRDataURL(content string) (string, error) {
	png, err := qrcode.Encode(content, qrcode.Medium, 256)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	buf.WriteString("data:image/png;base64,")
	buf.WriteString(base64.StdEncoding.EncodeToString(png))
	return buf.String(), nil
}
