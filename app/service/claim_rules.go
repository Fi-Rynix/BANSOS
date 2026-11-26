package service

import (
	"slices"

	"bansos-backend/app/model"
	"bansos-backend/helper"
)

var claimTransitions = map[string][]string{
	model.ClaimPending: {
		model.ClaimApproved, model.ClaimRejected, model.ClaimCancelled,
		model.ClaimExpired, model.ClaimWaitlisted,
	},
	model.ClaimWaitlisted: {
		model.ClaimPending, model.ClaimApproved, model.ClaimRejected,
		model.ClaimCancelled, model.ClaimExpired,
	},
	model.ClaimApproved: {
		model.ClaimCancelled, model.ClaimExpired,
	},
}

func IsValidClaimStatus(status string) bool {
	switch status {
	case model.ClaimPending, model.ClaimApproved, model.ClaimRejected,
		model.ClaimCancelled, model.ClaimExpired, model.ClaimWaitlisted:
		return true
	}
	return false
}

func CanTransitionClaim(from, to string) bool {
	return slices.Contains(claimTransitions[from], to)
}

func IsFinalClaimStatus(status string) bool {
	return len(claimTransitions[status]) == 0
}

func ClaimHoldsSlot(status string) bool {
	return status == model.ClaimPending || status == model.ClaimApproved
}

func TransitionReleasesSlot(from, to string) bool {
	return ClaimHoldsSlot(from) && !ClaimHoldsSlot(to)
}

func TransitionTakesSlot(from, to string) bool {
	return !ClaimHoldsSlot(from) && ClaimHoldsSlot(to)
}

func CheckClaimTransition(from, to string) error {
	if !IsValidClaimStatus(to) {
		return helper.BadRequest("status klaim tidak dikenal")
	}
	if !CanTransitionClaim(from, to) {
		return helper.Conflict("klaim berstatus " + from + " tidak dapat diubah menjadi " + to)
	}
	return nil
}

func CheckClaimOwnership(claim *model.Claim, userID int) error {
	if claim.UserID != userID {
		return helper.NotFound("klaim tidak ditemukan")
	}
	return nil
}

func CheckUserCancel(claim *model.Claim, userID int) error {
	if err := CheckClaimOwnership(claim, userID); err != nil {
		return err
	}
	return CheckClaimTransition(claim.Status, model.ClaimCancelled)
}
