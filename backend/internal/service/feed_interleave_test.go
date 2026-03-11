// feed_interleave_test.go — Unit tests for feed interleaving and ad filtering logic
package service

import (
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

func makeNotices(n int) []model.NoticeItem {
	notices := make([]model.NoticeItem, n)
	for i := range notices {
		notices[i] = model.NoticeItem{SEQ: i + 1, Subject: "Notice"}
	}
	return notices
}

func makeAds(n int) []model.AdItem {
	ads := make([]model.AdItem, n)
	for i := range ads {
		ads[i] = model.AdItem{MASeq: 100 + i, MAName: "Ad"}
	}
	return ads
}

func TestInterleaveFeed(t *testing.T) {
	tests := []struct {
		name          string
		notices       []model.NoticeItem
		ads           []model.AdItem
		wantTotal     int
		wantAdCount   int
		wantAdAfter4  bool // first ad should appear after index 4 (positions 0-3 are notices)
	}{
		{
			name:         "empty notices and ads",
			notices:      nil,
			ads:          nil,
			wantTotal:    0,
			wantAdCount:  0,
			wantAdAfter4: false,
		},
		{
			name:         "notices only no ads",
			notices:      makeNotices(8),
			ads:          nil,
			wantTotal:    8,
			wantAdCount:  0,
			wantAdAfter4: false,
		},
		{
			name:         "fewer than 4 notices no ads inserted",
			notices:      makeNotices(3),
			ads:          makeAds(5),
			wantTotal:    3,
			wantAdCount:  0,
			wantAdAfter4: false,
		},
		{
			name:         "exactly 4 notices one ad inserted",
			notices:      makeNotices(4),
			ads:          makeAds(3),
			wantTotal:    5,
			wantAdCount:  1,
			wantAdAfter4: true,
		},
		{
			name:         "8 notices two ads inserted",
			notices:      makeNotices(8),
			ads:          makeAds(5),
			wantTotal:    10,
			wantAdCount:  2,
			wantAdAfter4: true,
		},
		{
			name:         "more slots than ads",
			notices:      makeNotices(12),
			ads:          makeAds(1),
			wantTotal:    13,
			wantAdCount:  1,
			wantAdAfter4: true,
		},
		{
			name:         "ads but no notices",
			notices:      nil,
			ads:          makeAds(3),
			wantTotal:    0,
			wantAdCount:  0,
			wantAdAfter4: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			items := interleaveFeed(tt.notices, tt.ads)

			if len(items) != tt.wantTotal {
				t.Errorf("total items = %d, want %d", len(items), tt.wantTotal)
			}

			adCount := 0
			for _, item := range items {
				if item.Type == "ad" {
					adCount++
				}
			}
			if adCount != tt.wantAdCount {
				t.Errorf("ad count = %d, want %d", adCount, tt.wantAdCount)
			}

			if tt.wantAdAfter4 && len(items) > 4 {
				if items[4].Type != "ad" {
					t.Errorf("item at index 4 should be ad, got %q", items[4].Type)
				}
			}
		})
	}
}

func TestInterleaveFeedAdPositions(t *testing.T) {
	notices := makeNotices(12)
	ads := makeAds(10)

	items := interleaveFeed(notices, ads)

	// With 12 notices, ads should be at positions after every 4th notice.
	// Notices at indices 0,1,2,3 -> ad at 4
	// Notices at indices 5,6,7,8 -> ad at 9
	// Notices at indices 10,11,12,13 -> ad at 14
	expectedAdPositions := []int{4, 9, 14}

	for _, pos := range expectedAdPositions {
		if pos >= len(items) {
			t.Fatalf("expected ad at position %d but items length is %d", pos, len(items))
		}
		if items[pos].Type != "ad" {
			t.Errorf("expected ad at position %d, got %q", pos, items[pos].Type)
		}
	}
}

func TestFilterAds(t *testing.T) {
	ads := makeAds(5) // MASeq: 100, 101, 102, 103, 104

	tests := []struct {
		name       string
		excludeIDs []int
		wantCount  int
		wantFirst  int // MASeq of first remaining ad
	}{
		{"no exclusions", nil, 5, 100},
		{"empty exclusion slice", []int{}, 5, 100},
		{"exclude one", []int{102}, 4, 100},
		{"exclude first", []int{100}, 4, 101},
		{"exclude multiple", []int{100, 102, 104}, 2, 101},
		{"exclude all", []int{100, 101, 102, 103, 104}, 0, 0},
		{"exclude nonexistent", []int{999}, 5, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := filterAds(ads, tt.excludeIDs)

			if len(result) != tt.wantCount {
				t.Errorf("filterAds count = %d, want %d", len(result), tt.wantCount)
			}

			if tt.wantCount > 0 && result[0].MASeq != tt.wantFirst {
				t.Errorf("first ad MASeq = %d, want %d", result[0].MASeq, tt.wantFirst)
			}
		})
	}
}

func TestFilterAdsDoesNotMutateInput(t *testing.T) {
	ads := makeAds(3)
	original := make([]model.AdItem, len(ads))
	copy(original, ads)

	_ = filterAds(ads, []int{101})

	for i, ad := range ads {
		if ad.MASeq != original[i].MASeq {
			t.Errorf("filterAds mutated input slice at index %d", i)
		}
	}
}
