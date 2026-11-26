package service

import (
	"errors"
	"testing"

	"bansos-backend/app/model"
	"bansos-backend/helper"
)

func TestCanTransitionClaim(t *testing.T) {
	cases := []struct {
		from, to string
		want     bool
	}{
		{model.ClaimPending, model.ClaimApproved, true},
		{model.ClaimPending, model.ClaimWaitlisted, true},
		{model.ClaimWaitlisted, model.ClaimPending, true},
		{model.ClaimApproved, model.ClaimCancelled, true},
		{model.ClaimApproved, model.ClaimPending, false},
		{model.ClaimRejected, model.ClaimApproved, false},
		{model.ClaimCancelled, model.ClaimPending, false},
		{model.ClaimExpired, model.ClaimApproved, false},
		{model.ClaimPending, model.ClaimPending, false},
	}

	for _, tc := range cases {
		if got := CanTransitionClaim(tc.from, tc.to); got != tc.want {
			t.Errorf("CanTransitionClaim(%s, %s) = %v, want %v", tc.from, tc.to, got, tc.want)
		}
	}
}

func TestIsFinalClaimStatus(t *testing.T) {
	for _, s := range []string{model.ClaimRejected, model.ClaimCancelled, model.ClaimExpired} {
		if !IsFinalClaimStatus(s) {
			t.Errorf("%s seharusnya final", s)
		}
	}
	for _, s := range []string{model.ClaimPending, model.ClaimApproved, model.ClaimWaitlisted} {
		if IsFinalClaimStatus(s) {
			t.Errorf("%s seharusnya belum final", s)
		}
	}
}

func TestSlotAccounting(t *testing.T) {
	if !TransitionReleasesSlot(model.ClaimPending, model.ClaimRejected) {
		t.Error("pending ke rejected harus melepas slot")
	}
	if !TransitionReleasesSlot(model.ClaimApproved, model.ClaimCancelled) {
		t.Error("approved ke cancelled harus melepas slot")
	}
	if TransitionReleasesSlot(model.ClaimWaitlisted, model.ClaimExpired) {
		t.Error("waitlisted tidak memegang slot, tidak ada yang dilepas")
	}
	if TransitionReleasesSlot(model.ClaimPending, model.ClaimApproved) {
		t.Error("pending ke approved tetap memegang slot")
	}
	if !TransitionTakesSlot(model.ClaimWaitlisted, model.ClaimPending) {
		t.Error("promosi waitlist harus mengambil slot")
	}
}

func TestCheckClaimTransitionErrors(t *testing.T) {
	var appErr *helper.AppError

	err := CheckClaimTransition(model.ClaimPending, "unknown")
	if !errors.As(err, &appErr) || appErr.Code != helper.CodeBadRequest {
		t.Fatalf("status tidak dikenal harus BAD_REQUEST, dapat %v", err)
	}

	err = CheckClaimTransition(model.ClaimRejected, model.ClaimApproved)
	if !errors.As(err, &appErr) || appErr.Code != helper.CodeConflict {
		t.Fatalf("transisi terlarang harus CONFLICT, dapat %v", err)
	}

	if err := CheckClaimTransition(model.ClaimPending, model.ClaimApproved); err != nil {
		t.Fatalf("transisi sah tidak boleh error: %v", err)
	}
}

func TestCheckUserCancel(t *testing.T) {
	var appErr *helper.AppError
	claim := &model.Claim{ID: 1, UserID: 7, Status: model.ClaimPending}

	err := CheckUserCancel(claim, 8)
	if !errors.As(err, &appErr) || appErr.Code != helper.CodeNotFound {
		t.Fatalf("klaim milik orang lain harus NOT_FOUND, dapat %v", err)
	}

	if err := CheckUserCancel(claim, 7); err != nil {
		t.Fatalf("pemilik boleh membatalkan klaim pending: %v", err)
	}

	claim.Status = model.ClaimRejected
	err = CheckUserCancel(claim, 7)
	if !errors.As(err, &appErr) || appErr.Code != helper.CodeConflict {
		t.Fatalf("klaim rejected tidak bisa dibatalkan, dapat %v", err)
	}
}
