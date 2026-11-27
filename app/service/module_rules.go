package service

import (
	"strings"

	"bansos-backend/app/model"
)

func NormalizeModuleCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func NormalizeCategory(category string) string {
	return strings.ToLower(strings.TrimSpace(category))
}

// ResolveModuleStatus menentukan filter status katalog. Hanya pemegang
// module:manage yang boleh melihat modul nonaktif.
func ResolveModuleStatus(requested string, canManage bool) (string, bool) {
	if requested == "" {
		return model.ModuleStatusActive, true
	}
	if requested != model.ModuleStatusActive && !canManage {
		return "", false
	}
	return requested, true
}

func NormalizeCreateModule(req *model.CreateModuleRequest) {
	req.Code = NormalizeModuleCode(req.Code)
	req.Title = strings.TrimSpace(req.Title)
	req.Category = NormalizeCategory(req.Category)
	req.Description = strings.TrimSpace(req.Description)
}

func NormalizeUpdateModule(req *model.UpdateModuleRequest) {
	req.Code = NormalizeModuleCode(req.Code)
	req.Title = strings.TrimSpace(req.Title)
	req.Category = NormalizeCategory(req.Category)
	req.Description = strings.TrimSpace(req.Description)
}
