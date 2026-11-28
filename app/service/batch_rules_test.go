package service

import (
	"testing"
	"time"

	"bansos-backend/app/model"
)

func TestCanTransitionBatch(t *testing.T) {
	allowed := [][2]string{
		{"draft", "scheduled"},
		{"scheduled", "draft"},
		{"scheduled", "open"},
		{"scheduled", "closed"},
		{"open", "closed"},
	}
	for _, tc := range allowed {
		if !CanTransitionBatch(tc[0], tc[1]) {
			t.Errorf("%s -> %s seharusnya boleh", tc[0], tc[1])
		}
	}

	denied := [][2]string{
		{"draft", "open"},
		{"draft", "closed"},
		{"open", "scheduled"},
		{"open", "draft"},
		{"closed", "open"},
		{"closed", "draft"},
	}
	for _, tc := range denied {
		if CanTransitionBatch(tc[0], tc[1]) {
			t.Errorf("%s -> %s seharusnya ditolak", tc[0], tc[1])
		}
	}
}

func TestCheckBatchTransitionTime(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)

	b := &model.Batch{Status: model.BatchDraft, OpensAt: past, ClosesAt: future}
	if CheckBatchTransition(b, model.BatchScheduled, now) == nil {
		t.Error("menjadwalkan batch yang opens_at-nya sudah lewat seharusnya ditolak")
	}

	b = &model.Batch{Status: model.BatchScheduled, OpensAt: past, ClosesAt: past}
	if CheckBatchTransition(b, model.BatchOpen, now) == nil {
		t.Error("membuka batch yang closes_at-nya sudah lewat seharusnya ditolak")
	}

	b = &model.Batch{Status: model.BatchScheduled, OpensAt: future, ClosesAt: future.Add(time.Hour)}
	if err := CheckBatchTransition(b, model.BatchOpen, now); err != nil {
		t.Errorf("membuka lebih awal seharusnya boleh: %v", err)
	}

	if CheckBatchTransition(&model.Batch{Status: model.BatchOpen}, model.BatchOpen, now) == nil {
		t.Error("status yang sama seharusnya ditolak")
	}
}

func TestCheckBatchWindow(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	if CheckBatchWindow(now.Add(time.Hour), now, now) == nil {
		t.Error("closes_at sebelum opens_at seharusnya ditolak")
	}
	if CheckBatchWindow(now.Add(-2*time.Hour), now.Add(-time.Hour), now) == nil {
		t.Error("closes_at di masa lalu seharusnya ditolak")
	}
	if err := CheckBatchWindow(now, now.Add(time.Hour), now); err != nil {
		t.Errorf("jendela valid ditolak: %v", err)
	}
}

func TestCheckBatchQuota(t *testing.T) {
	if CheckBatchQuota(2, 3) == nil {
		t.Error("kuota di bawah allocated seharusnya ditolak")
	}
	if CheckBatchQuota(3, 3) != nil {
		t.Error("kuota sama dengan allocated seharusnya boleh")
	}
}

func TestIsBatchAcceptingClaims(t *testing.T) {
	now := time.Date(2026, 11, 27, 10, 0, 0, 0, time.UTC)
	open := &model.Batch{Status: model.BatchOpen, OpensAt: now.Add(-time.Hour), ClosesAt: now.Add(time.Hour)}
	if !IsBatchAcceptingClaims(open, now) {
		t.Error("batch open di dalam jendela seharusnya menerima klaim")
	}
	if IsBatchAcceptingClaims(&model.Batch{Status: model.BatchOpen, OpensAt: now.Add(time.Minute), ClosesAt: now.Add(time.Hour)}, now) {
		t.Error("batch yang belum sampai opens_at seharusnya belum menerima klaim")
	}
	if IsBatchAcceptingClaims(&model.Batch{Status: model.BatchOpen, OpensAt: now.Add(-time.Hour), ClosesAt: now}, now) {
		t.Error("batch yang sudah sampai closes_at seharusnya tidak menerima klaim")
	}
	if IsBatchAcceptingClaims(&model.Batch{Status: model.BatchScheduled, OpensAt: now.Add(-time.Hour), ClosesAt: now.Add(time.Hour)}, now) {
		t.Error("batch scheduled seharusnya tidak menerima klaim")
	}
}
