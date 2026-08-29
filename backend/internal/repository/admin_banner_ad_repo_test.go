package repository

import (
	"reflect"
	"testing"
)

func TestRemovedURLs(t *testing.T) {
	previousURLs := []string{"kept.png", "removed.png", "removed.png"}
	currentURLs := []string{"kept.png", "new.png"}

	got := removedURLs(previousURLs, currentURLs)
	want := []string{"removed.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("removedURLs() = %v, want %v", got, want)
	}
}

func TestBannerAdPeriodsOverlap(t *testing.T) {
	date := func(value string) *string {
		return &value
	}

	tests := []struct {
		name           string
		existingStart  *string
		existingEnd    *string
		candidateStart *string
		candidateEnd   *string
		want           bool
	}{
		{
			name: "both periods are unbounded",
			want: true,
		},
		{
			name:           "unbounded existing start ends before candidate starts",
			existingEnd:    date("2026-01-31"),
			candidateStart: date("2026-02-01"),
			candidateEnd:   date("2026-02-28"),
			want:           false,
		},
		{
			name:           "unbounded existing start overlaps candidate",
			existingEnd:    date("2026-03-31"),
			candidateStart: date("2026-02-01"),
			candidateEnd:   date("2026-02-28"),
			want:           true,
		},
		{
			name:           "unbounded existing end overlaps candidate",
			existingStart:  date("2026-01-01"),
			candidateStart: date("2026-02-01"),
			candidateEnd:   date("2026-02-28"),
			want:           true,
		},
		{
			name:           "existing period ends before unbounded candidate starts",
			existingStart:  date("2026-01-01"),
			existingEnd:    date("2026-01-31"),
			candidateStart: date("2026-02-01"),
			want:           false,
		},
		{
			name:           "bounded periods overlap",
			existingStart:  date("2026-01-01"),
			existingEnd:    date("2026-01-31"),
			candidateStart: date("2026-01-15"),
			candidateEnd:   date("2026-02-28"),
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := bannerAdPeriodsOverlap(
				tt.existingStart,
				tt.existingEnd,
				tt.candidateStart,
				tt.candidateEnd,
			)
			if got != tt.want {
				t.Fatalf("bannerAdPeriodsOverlap() = %v, want %v", got, tt.want)
			}
		})
	}
}
