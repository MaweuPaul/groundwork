package service

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func day(s string) time.Time {
	t, _ := time.Parse(time.DateOnly, s)
	return t
}

func TestSeriesWindows(t *testing.T) {
	tests := []struct {
		name             string
		start, end       string
		window, interval int
		want             []string
	}{
		{"1-month windows every 3 months", "2025-01-01", "2025-05-31", 1, 3,
			[]string{"2025-01-01/2025-01-31", "2025-04-01/2025-04-30"}},
		{"back-to-back quarters", "2025-01-01", "2025-12-31", 3, 3,
			[]string{"2025-01-01/2025-03-31", "2025-04-01/2025-06-30", "2025-07-01/2025-09-30", "2025-10-01/2025-12-31"}},
		{"last window cut short at the end date", "2025-01-01", "2025-05-15", 3, 3,
			[]string{"2025-01-01/2025-03-31", "2025-04-01/2025-05-15"}},
		{"range shorter than one window", "2025-01-01", "2025-01-10", 3, 3,
			[]string{"2025-01-01/2025-01-10"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SeriesWindows(day(tt.start), day(tt.end), tt.window, tt.interval)
			if len(got) != len(tt.want) {
				t.Fatalf("got %d windows, want %d", len(got), len(tt.want))
			}
			for i, w := range got {
				if s := w[0].Format(time.DateOnly) + "/" + w[1].Format(time.DateOnly); s != tt.want[i] {
					t.Errorf("window %d = %s, want %s", i, s, tt.want[i])
				}
			}
		})
	}
}

func seriesInput() CreateSeriesInput {
	return CreateSeriesInput{CreateSnapshotInput: validInput(), WindowMonths: 1, IntervalMonths: 3}
}

func TestCreateSeriesQueuesAllWindows(t *testing.T) {
	snaps := &fakeSnapshotRepo{id: uuid.New()}
	svc := newTestSnapshotService(snaps, &fakeMethodologyRepo{m: ndvi})
	in := seriesInput()
	in.DateStart, in.DateEnd = "2024-01-01", "2025-12-31" // longer than a year is fine for a series

	ref, err := svc.CreateSeries(in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Windows != 8 || len(snaps.gotSeries.Windows) != 8 {
		t.Errorf("windows = %d, want 8", ref.Windows)
	}
	if snaps.gotSeries.Template.MaxSceneCloudPct != 20 || snaps.gotSeries.Template.CloudScoreThreshold != 0.6 {
		t.Errorf("cloud defaults not applied: %+v", snaps.gotSeries.Template)
	}
}

func TestCreateSeriesRejectsBadInput(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*CreateSeriesInput)
	}{
		{"zero window", func(in *CreateSeriesInput) { in.WindowMonths = 0 }},
		{"interval over a year", func(in *CreateSeriesInput) { in.IntervalMonths = 13 }},
		{"too many windows", func(in *CreateSeriesInput) {
			in.DateStart, in.DateEnd, in.IntervalMonths = "2020-01-01", "2026-10-31", 1
		}},
		{"end in the future", func(in *CreateSeriesInput) { in.DateEnd = "2026-12-01" }},
		{"bad cloud setting", func(in *CreateSeriesInput) { in.MaxSceneCloudPct = ptr(0) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snaps := &fakeSnapshotRepo{}
			svc := newTestSnapshotService(snaps, &fakeMethodologyRepo{m: ndvi})
			in := seriesInput()
			tt.modify(&in)
			_, err := svc.CreateSeries(in)
			var invalid *ValidationError
			if !errors.As(err, &invalid) {
				t.Fatalf("err = %v, want *ValidationError", err)
			}
			if snaps.calls != 0 {
				t.Error("repository was called for an invalid request")
			}
		})
	}
}
