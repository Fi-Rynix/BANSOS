package service

import (
	"strconv"
	"strings"
	"unicode"

	"bansos-backend/app/model"
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 72
)

// weakPasswords mengikuti pendekatan api-students: daftar statis password yang
// paling sering muncul. Password seed admin BANSOS sengaja tidak dimasukkan.
var weakPasswords = map[string]bool{
	"password":      true,
	"password1":     true,
	"password12":    true,
	"password123":   true,
	"password1234":  true,
	"12345678":      true,
	"123456789":     true,
	"1234567890":    true,
	"qwerty":        true,
	"qwerty123":     true,
	"qwerty1234":    true,
	"admin":         true,
	"admin123":      true,
	"admin1234":     true,
	"adminadmin":    true,
	"administrator": true,
	"root":          true,
	"root1234":      true,
	"letmein":       true,
	"letmein123":    true,
	"welcome":       true,
	"welcome1":      true,
	"welcome123":    true,
	"iloveyou":      true,
	"peserta":       true,
	"peserta123":    true,
	"peserta1234":   true,
	"pengelola":     true,
	"pengelola123":  true,
	"bansos":        true,
	"bansos123":     true,
	"bansos1234":    true,
	"bansos2026":    true,
	"modul123":      true,
	"modul1234":     true,
	"modul12345":    true,
	"praktikum":     true,
	"praktikum123":  true,
	"backend":       true,
	"backend123":    true,
	"backend1234":   true,
}

func isWeakPassword(password string) bool {
	return weakPasswords[strings.ToLower(strings.TrimSpace(password))]
}

// PasswordStrengthReason mengembalikan alasan pertama password ditolak, string
// kosong berarti lolos. Dipanggil dari ValidateRegister sebelum menyimpan.
func PasswordStrengthReason(password string) string {
	if strings.TrimSpace(password) == "" {
		return "wajib diisi"
	}
	if len(password) < MinPasswordLength {
		return "minimal " + strconv.Itoa(MinPasswordLength) + " karakter"
	}
	if len(password) > MaxPasswordLength {
		return "maksimal " + strconv.Itoa(MaxPasswordLength) + " karakter"
	}

	var hasUpper, hasLower, hasDigit bool
	for _, r := range password {
		switch {
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	switch {
	case !hasUpper:
		return "harus memuat huruf besar"
	case !hasLower:
		return "harus memuat huruf kecil"
	case !hasDigit:
		return "harus memuat angka"
	}

	if isWeakPassword(password) {
		return "terlalu umum, pilih password lain"
	}
	return ""
}

func NormalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func ValidateRegister(req model.RegisterRequest) map[string]string {
	errs := map[string]string{}

	name := strings.TrimSpace(req.Name)
	switch {
	case name == "":
		errs["name"] = "wajib diisi"
	case len(name) < 3:
		errs["name"] = "minimal 3 karakter"
	case len(name) > 100:
		errs["name"] = "maksimal 100 karakter"
	}

	if !isValidEmail(NormalizeEmail(req.Email)) {
		errs["email"] = "format email tidak valid"
	}

	if msg := PasswordStrengthReason(req.Password); msg != "" {
		errs["password"] = msg
	}
	return errs
}

func ValidateLogin(req model.LoginRequest) map[string]string {
	errs := map[string]string{}
	if !isValidEmail(NormalizeEmail(req.Email)) {
		errs["email"] = "format email tidak valid"
	}
	if req.Password == "" {
		errs["password"] = "wajib diisi"
	}
	return errs
}

func ValidateUpdateRole(req model.UpdateRoleRequest) map[string]string {
	if !model.IsKnownRole(req.Role) {
		return map[string]string{"role": "harus salah satu dari: admin, pengelola, peserta"}
	}
	return nil
}

func isValidEmail(email string) bool {
	if strings.ContainsAny(email, " \t\n\r") {
		return false
	}
	at := strings.Index(email, "@")
	if at < 1 {
		return false
	}
	domain := email[at+1:]
	dot := strings.Index(domain, ".")
	return dot > 0 && dot < len(domain)-1
}
