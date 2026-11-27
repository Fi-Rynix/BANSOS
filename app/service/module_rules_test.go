package service

import (
	"testing"

	"bansos-backend/app/model"
)

func TestNormalizeModuleCode(t *testing.T) {
	if got := NormalizeModuleCode("  go-101 "); got != "GO-101" {
		t.Fatalf("got %q, want GO-101", got)
	}
}

func TestResolveModuleStatus(t *testing.T) {
	cases := []struct {
		requested string
		canManage bool
		want      string
		ok        bool
	}{
		{"", false, model.ModuleStatusActive, true},
		{"", true, model.ModuleStatusActive, true},
		{"active", false, model.ModuleStatusActive, true},
		{"inactive", false, "", false},
		{"all", false, "", false},
		{"inactive", true, model.ModuleStatusInactive, true},
		{"all", true, model.ModuleStatusAll, true},
	}
	for _, tc := range cases {
		got, ok := ResolveModuleStatus(tc.requested, tc.canManage)
		if got != tc.want || ok != tc.ok {
			t.Errorf("ResolveModuleStatus(%q, %v) = (%q, %v), want (%q, %v)",
				tc.requested, tc.canManage, got, ok, tc.want, tc.ok)
		}
	}
}

func TestNormalizeCreateModule(t *testing.T) {
	req := model.CreateModuleRequest{Code: " sql-201 ", Title: " Lanjut ", Category: " Database ", Description: " x "}
	NormalizeCreateModule(&req)
	if req.Code != "SQL-201" || req.Title != "Lanjut" || req.Category != "database" || req.Description != "x" {
		t.Fatalf("hasil normalisasi salah: %+v", req)
	}
}
