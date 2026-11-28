package service

import (
	"testing"

	"bansos-backend/app/model"
)

func modelTestRegisterOK() model.RegisterRequest {
	return model.RegisterRequest{Name: "Ihya Azzaky", Email: "ihya@bansos.test", Password: "RahasiaKuat1"}
}

func modelTestRegisterEmpty() model.RegisterRequest {
	return model.RegisterRequest{}
}

func modelTestRole(role string) model.UpdateRoleRequest {
	return model.UpdateRoleRequest{Role: role}
}

func TestPasswordStrengthReason(t *testing.T) {
	cases := []struct {
		password string
		want     string
	}{
		{"", "wajib diisi"},
		{"Ab1", "minimal 8 karakter"},
		{"abcdefg1", "harus memuat huruf besar"},
		{"ABCDEFG1", "harus memuat huruf kecil"},
		{"Abcdefgh", "harus memuat angka"},
		{"Password1", "terlalu umum, pilih password lain"},
		{" ", "wajib diisi"},
		{"RahasiaKuat1", ""},
	}
	for _, c := range cases {
		if got := PasswordStrengthReason(c.password); got != c.want {
			t.Errorf("PasswordStrengthReason(%q) = %q, want %q", c.password, got, c.want)
		}
	}
}

func TestWeakPasswordsListJanganMemuatSeedAdmin(t *testing.T) {
	if isWeakPassword("BansosAdmin2026") {
		t.Fatal("password seed admin tidak boleh masuk daftar lemah")
	}
	if isWeakPassword("bansosadmin2026") {
		t.Fatal("pembanding lemah harus case insensitive dan bukan seed admin")
	}
}

func TestValidateRegister(t *testing.T) {
	errs := ValidateRegister(modelTestRegisterOK())
	if len(errs) != 0 {
		t.Fatalf("registrasi valid ditolak: %v", errs)
	}

	bad := ValidateRegister(modelTestRegisterEmpty())
	for _, key := range []string{"name", "email", "password"} {
		if bad[key] == "" {
			t.Errorf("field %s seharusnya punya pesan error, dapat %v", key, bad)
		}
	}
}

func TestValidateUpdateRole(t *testing.T) {
	if errs := ValidateUpdateRole(modelTestRole("peserta")); errs != nil {
		t.Errorf("role peserta seharusnya valid, dapat %v", errs)
	}
	if errs := ValidateUpdateRole(modelTestRole("superadmin")); errs["role"] == "" {
		t.Errorf("role tak dikenal seharusnya ditolak, dapat %v", errs)
	}
}

func TestNormalizeEmail(t *testing.T) {
	if got := NormalizeEmail("  Admin-BANSOS@Gmail.COM "); got != "admin-bansos@gmail.com" {
		t.Errorf("NormalizeEmail = %q", got)
	}
}
