package service

import (
	"slices"

	"bansos-backend/app/model"
	"bansos-backend/helper"
)

const (
	MinQuizScore = 0
	MaxQuizScore = 100
)

func CheckLessonExists(lessonSlug string, moduleLessons []string) error {
	if slices.Contains(moduleLessons, lessonSlug) {
		return nil
	}
	return helper.NotFound("lesson tidak ditemukan di modul ini")
}

func CheckQuizScore(score int) error {
	if score < MinQuizScore || score > MaxQuizScore {
		return helper.BadRequest("skor kuis harus di antara 0 dan 100")
	}
	return nil
}

func CountCompletedLessons(completed, moduleLessons []string) int {
	done := make(map[string]struct{}, len(completed))
	for _, slug := range completed {
		done[slug] = struct{}{}
	}

	count := 0
	for _, slug := range moduleLessons {
		if _, ok := done[slug]; ok {
			count++
		}
	}
	return count
}

func IsModuleCompleted(completed, moduleLessons []string) bool {
	return len(moduleLessons) > 0 && CountCompletedLessons(completed, moduleLessons) == len(moduleLessons)
}

func SummarizeProgress(p model.Progress, moduleLessons []string) model.ProgressSummary {
	total := len(moduleLessons)
	done := CountCompletedLessons(p.CompletedLessons, moduleLessons)

	percent := 0
	if total > 0 {
		percent = done * 100 / total
	}

	return model.ProgressSummary{
		Progress:     p,
		TotalLessons: total,
		Percent:      percent,
		Completed:    total > 0 && done == total,
	}
}
