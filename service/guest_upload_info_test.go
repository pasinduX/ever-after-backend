package service

import (
	"strings"
	"testing"
	"time"
)

func TestParseClientCaptureTime(t *testing.T) {
	now := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)

	got := ParseClientCaptureTime("2026:10:31 18:22:05", "+05:30", now)
	if got == nil || !got.Equal(time.Date(2026, 10, 31, 12, 52, 5, 0, time.UTC)) {
		t.Errorf("with offset = %v, want 12:52:05 UTC", got)
	}
	// Without an offset it reads like EXIF inside a file: in the server's zone.
	got = ParseClientCaptureTime("2026:10:31 18:22:05", "", now)
	if got == nil || got.Location() != time.Local || got.Hour() != 18 {
		t.Errorf("without offset = %v, want 18:22 local", got)
	}
	for _, bad := range []string{"", "yesterday", "1970:01:01 00:00:00", "2030:01:01 00:00:00", "2026-10-31T18:22:05Z"} {
		if got := ParseClientCaptureTime(bad, "", now); got != nil {
			t.Errorf("ParseClientCaptureTime(%q) = %v, want nil", bad, got)
		}
	}
}

func TestCleanGuestName(t *testing.T) {
	if got := CleanGuestName("  Aunt   Nimali \n"); got != "Aunt Nimali" {
		t.Errorf("got %q", got)
	}
	if got := CleanGuestName(strings.Repeat("a", 100)); len([]rune(got)) > 61 {
		t.Errorf("name not capped: %d runes", len([]rune(got)))
	}
}
