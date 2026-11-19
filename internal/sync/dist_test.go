package sync

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/vasfvitor/nanci/internal/sefaz"
	"github.com/vasfvitor/nanci/internal/syncstate"
)

func TestDistBatch(t *testing.T) {
	docs := []sefaz.DocZip{
		{NSU: 11, Schema: "doc"},
		{NSU: 12, Schema: "event"},
	}
	tests := []struct {
		name       string
		resp       sefaz.DistResult
		wantCursor int64
		wantItems  int
		wantReason syncstate.SyncStopReason // "" means not done
	}{
		{"138 with more to fetch", sefaz.DistResult{CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 12, MaxNSU: 20, Docs: docs}, 12, 2, ""},
		{"138 caught up", sefaz.DistResult{CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 12, MaxNSU: 12, Docs: docs}, 12, 2, syncstate.SyncStopReasonCaughtUp},
		{"137 moves forward", sefaz.DistResult{CStat: sefaz.CStatNenhumDocumento, UltNSU: 15, MaxNSU: 15}, 15, 0, syncstate.SyncStopReasonCaughtUp},
		{"137 never moves back", sefaz.DistResult{CStat: sefaz.CStatNenhumDocumento, UltNSU: 8, MaxNSU: 8}, 10, 0, syncstate.SyncStopReasonCaughtUp},
		{"656 never moves back", sefaz.DistResult{CStat: sefaz.CStatConsumoIndevido, UltNSU: 4}, 10, 0, syncstate.SyncStopReasonConsumoIndevido},
		{"656 moves forward", sefaz.DistResult{CStat: sefaz.CStatConsumoIndevido, UltNSU: 12}, 12, 0, syncstate.SyncStopReasonConsumoIndevido},
	}
	isEvent := func(schema string) bool { return schema == "event" }
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch, err := distBatch(context.Background(), discardLogger(), tt.resp, 10, isEvent, "NF-e")
			if err != nil {
				t.Fatal(err)
			}
			if batch.NextCursor != tt.wantCursor || len(batch.Items) != tt.wantItems {
				t.Errorf("(cursor, items) = (%d, %d), want (%d, %d)", batch.NextCursor, len(batch.Items), tt.wantCursor, tt.wantItems)
			}
			if tt.wantItems > 0 && (batch.Items[0].IsEvent || !batch.Items[1].IsEvent) {
				t.Errorf("items = %+v, want the second one as the event", batch.Items)
			}
			if batch.UltNSU != tt.resp.UltNSU || batch.MaxNSU != tt.resp.MaxNSU {
				t.Errorf("(ultNSU, maxNSU) = (%d, %d), want the response's", batch.UltNSU, batch.MaxNSU)
			}

			if tt.wantReason == "" {
				if batch.Done || batch.WaitUntil != nil {
					t.Errorf("batch = %+v, want more to fetch", batch)
				}
				return
			}
			if !batch.Done || batch.StopReason != tt.wantReason {
				t.Errorf("(done, reason) = (%v, %s), want (true, %s)", batch.Done, batch.StopReason, tt.wantReason)
			}
			if batch.WaitUntil == nil || time.Until(*batch.WaitUntil) < 59*time.Minute {
				t.Errorf("WaitUntil = %v, want an hour from now", batch.WaitUntil)
			}
		})
	}

	_, err := distBatch(context.Background(), discardLogger(), sefaz.DistResult{CStat: 593, XMotivo: "rejeitada"}, 10, isEvent, "CT-e")
	var rejection *sefaz.RejectionError
	if !errors.As(err, &rejection) || rejection.CStat != 593 {
		t.Errorf("distBatch error = %v, want the 593 rejection", err)
	}
}

func TestDistBatchGaps(t *testing.T) {
	page := func(nsus ...int64) []sefaz.DocZip {
		docs := make([]sefaz.DocZip, 0, len(nsus))
		for _, nsu := range nsus {
			docs = append(docs, sefaz.DocZip{NSU: nsu, Schema: "doc"})
		}
		return docs
	}
	tests := []struct {
		name   string
		cursor int64
		resp   sefaz.DistResult
		want   []NSURange
	}{
		{"138 without holes", 10, sefaz.DistResult{CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 13, MaxNSU: 20, Docs: page(11, 12, 13)}, nil},
		{"138 with holes", 10, sefaz.DistResult{CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 16, MaxNSU: 20, Docs: page(12, 14)},
			[]NSURange{{11, 11, NSUGapBatchHole}, {13, 13, NSUGapBatchHole}, {15, 16, NSUGapBatchHole}}},
		{"138 from cursor 0 starts at the window", 0, sefaz.DistResult{CStat: sefaz.CStatDocumentoLocalizado, UltNSU: 52, MaxNSU: 60, Docs: page(50, 52)}, nil},
		{"137 jump is not a gap", 10, sefaz.DistResult{CStat: sefaz.CStatNenhumDocumento, UltNSU: 15, MaxNSU: 15}, nil},
		{"656 jump", 10, sefaz.DistResult{CStat: sefaz.CStatConsumoIndevido, UltNSU: 400}, []NSURange{{11, 400, NSUGapConsumoIndevido}}},
		{"656 behind the cursor", 10, sefaz.DistResult{CStat: sefaz.CStatConsumoIndevido, UltNSU: 4}, nil},
		{"656 from cursor 0", 0, sefaz.DistResult{CStat: sefaz.CStatConsumoIndevido, UltNSU: 400}, nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			batch, err := distBatch(context.Background(), discardLogger(), tt.resp, tt.cursor, func(string) bool { return false }, "NF-e")
			if err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(batch.Gaps, tt.want) {
				t.Errorf("gaps = %+v, want %+v", batch.Gaps, tt.want)
			}
		})
	}
}

func TestNSUGapsIgnoresNSUsOutsideTheRangeAndRepeats(t *testing.T) {
	got := nsuGaps(10, 15, []int64{9, 12, 12, 20, 15}, NSUGapBatchHole)
	want := []NSURange{{11, 11, NSUGapBatchHole}, {13, 14, NSUGapBatchHole}}
	if !slices.Equal(got, want) {
		t.Errorf("gaps = %+v, want %+v", got, want)
	}
}

func TestDistIdle(t *testing.T) {
	now := time.Date(2026, 10, 7, 12, 0, 0, 0, time.UTC)
	daysAgo := func(days int) *time.Time {
		at := now.Add(-time.Duration(days) * 24 * time.Hour)
		return &at
	}
	almost45 := now.Add(-45*24*time.Hour + time.Minute)
	tests := []struct {
		name string
		last *time.Time
		want int
	}{
		{"44 days", daysAgo(44), 0},
		{"a minute short of 45 days", &almost45, 0},
		{"45 days", daysAgo(45), 45},
		{"90 days", daysAgo(90), 90},
		{"never synced", nil, 0},
		{"clock behind the last query", daysAgo(-1), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DistIdle(tt.last, now); got != tt.want {
				t.Errorf("DistIdle = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestRequestsPerHour(t *testing.T) {
	for source, want := range map[syncstate.SyncSource]int{
		syncstate.SyncSourceNFe:  20,
		syncstate.SyncSourceCTe:  20,
		syncstate.SyncSourceNFSe: 0,
	} {
		if got := requestsPerHour(source); got != want {
			t.Errorf("requestsPerHour(%s) = %d, want %d", source, got, want)
		}
	}
}

func TestCheckTpAmb(t *testing.T) {
	tests := []struct {
		xml, want    string
		wantWarnings int
	}{
		{"", "1", 0},
		{"1", "1", 0},
		{"2", "2", 1},
		{"9", "1", 1},
	}
	for _, tt := range tests {
		var warnings []string
		if got := CheckTpAmb(tt.xml, "1", &warnings); got != tt.want || len(warnings) != tt.wantWarnings {
			t.Errorf("CheckTpAmb(%q, 1) = %q with %v, want %q with %d warnings", tt.xml, got, warnings, tt.want, tt.wantWarnings)
		}
	}
}
