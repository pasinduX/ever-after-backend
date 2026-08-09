package service

import (
	"testing"

	"github.com/storyvows/backend/dto"
)

func intPtr(v int) *int { return &v }

func TestCategoryFromAnalysis(t *testing.T) {
	cases := []struct {
		name       string
		eventType  string
		basicScene string
		faceCount  *int
		want       dto.UploadCategory
	}{
		{"ceremony event", "ceremony", "people", intPtr(8), dto.CategoryCeremony},
		{"reception event", "reception", "people", intPtr(12), dto.CategoryDancing},
		{"getting ready", "getting_ready", "people", intPtr(2), dto.CategoryCandid},
		{"pre-shoot couple", "pre_shoot", "people", intPtr(2), dto.CategoryCandid},
		{"pre-shoot group", "pre_shoot", "people", intPtr(6), dto.CategoryFamily},
		{"case insensitive", "CEREMONY", "", nil, dto.CategoryCeremony},
		{"padded value", "  reception  ", "", nil, dto.CategoryDancing},
		{"scene fallback ceremony", "other", "ceremony", nil, dto.CategoryCeremony},
		{"scene fallback group", "other", "people", intPtr(5), dto.CategoryFamily},
		{"scene fallback couple", "", "people", intPtr(2), dto.CategoryCandid},
		{"venue detail", "other", "venue", nil, dto.CategoryOther},
		{"nothing known", "", "", nil, dto.CategoryOther},
		{"nil face count on people", "other", "people", nil, dto.CategoryCandid},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := categoryFromAnalysis(tc.eventType, tc.basicScene, tc.faceCount)
			if got != tc.want {
				t.Fatalf("categoryFromAnalysis(%q, %q, %v) = %q, want %q",
					tc.eventType, tc.basicScene, tc.faceCount, got, tc.want)
			}
		})
	}
}

// The regression this guards: before event_type was mapped onto Category every
// upload stayed CategoryOther, so assignActsByCategory put the entire wedding
// into a single ANTICIPATION act.
func TestAssignActsByCategorySplitsAcrossActs(t *testing.T) {
	mk := func(eventType string, faces int) *dto.Upload {
		cat := categoryFromAnalysis(eventType, "people", intPtr(faces))
		return &dto.Upload{
			ID:       eventType + string(rune('a'+faces)),
			Category: cat,
			Analysis: dto.UploadAnalysis{Category: cat},
		}
	}

	uploads := []*dto.Upload{
		mk("ceremony", 8),
		mk("ceremony", 4),
		mk("reception", 12),
		mk("pre_shoot", 6),
		mk("getting_ready", 2),
	}

	acts := assignActsByCategory("wedding-1", uploads)

	byLabel := map[dto.ActLabel]int{}
	for _, a := range acts {
		byLabel[a.Label] = len(a.PhotoIDs)
	}

	if len(acts) != 4 {
		t.Fatalf("expected 4 distinct acts, got %d (%v)", len(acts), byLabel)
	}
	if byLabel[dto.ActCeremony] != 2 {
		t.Errorf("ceremony act = %d photos, want 2", byLabel[dto.ActCeremony])
	}
	if byLabel[dto.ActCelebration] != 1 {
		t.Errorf("celebration act = %d photos, want 1", byLabel[dto.ActCelebration])
	}
	if byLabel[dto.ActFamilyBonds] != 1 {
		t.Errorf("family act = %d photos, want 1", byLabel[dto.ActFamilyBonds])
	}
	if byLabel[dto.ActAnticipation] != 1 {
		t.Errorf("anticipation act = %d photos, want 1", byLabel[dto.ActAnticipation])
	}
}
