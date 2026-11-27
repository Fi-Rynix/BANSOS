package service

import (
	"slices"
	"time"

	"bansos-backend/app/model"
	"bansos-backend/helper"
)

// Alur status batch:
// draft -> scheduled -> open -> closed
// scheduled bisa dikembalikan ke draft atau langsung ditutup (batal).
var batchTransitions = map[string][]string{
	model.BatchDraft:     {model.BatchScheduled},
	model.BatchScheduled: {model.BatchDraft, model.BatchOpen, model.BatchClosed},
	model.BatchOpen:      {model.BatchClosed},
}

func CanTransitionBatch(from, to string) bool {
	return slices.Contains(batchTransitions[from], to)
}

// IsBatchEditable: jadwal dan kuota hanya boleh diubah sebelum war dibuka.
func IsBatchEditable(status string) bool {
	return status == model.BatchDraft || status == model.BatchScheduled
}

// IsBatchVisible: batch draft hanya terlihat oleh pemegang batch:manage.
func IsBatchVisible(status string, canManage bool) bool {
	return canManage || status != model.BatchDraft
}

// IsBatchAcceptingClaims dipakai war slot: batch harus open dan waktu
// sekarang ada di dalam jendela opens_at sampai closes_at.
func IsBatchAcceptingClaims(b *model.Batch, now time.Time) bool {
	return b.Status == model.BatchOpen && !now.Before(b.OpensAt) && now.Before(b.ClosesAt)
}

func CheckBatchWindow(opensAt, closesAt, now time.Time) error {
	fields := map[string]string{}
	if !closesAt.After(opensAt) {
		fields["closes_at"] = "harus setelah opens_at"
	} else if !closesAt.After(now) {
		fields["closes_at"] = "harus di masa depan"
	}
	if len(fields) > 0 {
		return helper.Validation(fields)
	}
	return nil
}

func CheckBatchQuota(quota, allocated int) error {
	if quota < allocated {
		return helper.Validation(map[string]string{
			"quota": "tidak boleh lebih kecil dari slot yang sudah terisi",
		})
	}
	return nil
}

// CheckBatchTransition memvalidasi perpindahan status beserta syarat waktunya.
func CheckBatchTransition(b *model.Batch, to string, now time.Time) error {
	if b.Status == to {
		return helper.Conflict("batch sudah berstatus " + to)
	}
	if !CanTransitionBatch(b.Status, to) {
		return helper.Conflict("status batch tidak bisa diubah dari " + b.Status + " ke " + to)
	}
	switch to {
	case model.BatchScheduled:
		if !b.OpensAt.After(now) {
			return helper.Conflict("opens_at sudah lewat, ubah jadwal dulu sebelum menjadwalkan")
		}
	case model.BatchOpen:
		if !b.ClosesAt.After(now) {
			return helper.Conflict("closes_at sudah lewat, batch tidak bisa dibuka")
		}
	}
	return nil
}
