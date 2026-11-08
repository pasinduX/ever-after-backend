package service

import (
	"errors"
	"testing"
	"time"

	"github.com/storyvows/backend/dto"
	apperrors "github.com/storyvows/backend/errors"
	"github.com/storyvows/backend/functions"
	"github.com/storyvows/backend/integrations"
)

func TestAlbumAccessRules(t *testing.T) {
	svc := &WeddingService{cfg: &integrations.Secrets{JWTSecret: "s"}}
	token, _, _ := functions.SignAlbumAccess("s", "w1", time.Hour)
	otherToken, _, _ := functions.SignAlbumAccess("s", "w2", time.Hour)

	cases := []struct {
		name    string
		privacy dto.Privacy
		user    string
		token   string
		want    error
	}{
		{"public guest", dto.PrivacyPublic, "", "", nil},
		{"password guest without token", dto.PrivacyPasswordProtected, "", "", apperrors.ErrAlbumPasswordRequired},
		{"password guest with token", dto.PrivacyPasswordProtected, "", token, nil},
		{"password guest with another wedding's token", dto.PrivacyPasswordProtected, "", otherToken, apperrors.ErrAlbumPasswordRequired},
		{"private guest", dto.PrivacyPrivate, "", token, apperrors.ErrAlbumPrivate},
		{"private, someone else signed in", dto.PrivacyPrivate, "stranger", "", apperrors.ErrAlbumPrivate},
		{"private, the couple", dto.PrivacyPrivate, "owner", "", nil},
		{"password, the couple", dto.PrivacyPasswordProtected, "owner", "", nil},
	}
	for _, tc := range cases {
		w := &dto.Wedding{ID: "w1", OwnerID: "owner", Privacy: tc.privacy}
		if err := svc.albumAccessError(w, tc.user, tc.token); !errors.Is(err, tc.want) {
			t.Errorf("%s: got %v, want %v", tc.name, err, tc.want)
		}
	}
}
