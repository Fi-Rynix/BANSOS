package service

import (
	"testing"

	"bansos-backend/app/model"
)

var sampleLessons = []string{"intro", "variabel", "fungsi", "error"}

func TestCheckLessonExists(t *testing.T) {
	if err := CheckLessonExists("fungsi", sampleLessons); err != nil {
		t.Fatalf("lesson ada, tidak boleh error: %v", err)
	}
	if err := CheckLessonExists("goroutine", sampleLessons); err == nil {
		t.Fatal("lesson tidak ada harus error")
	}
}

func TestCheckQuizScore(t *testing.T) {
	for _, s := range []int{0, 50, 100} {
		if err := CheckQuizScore(s); err != nil {
			t.Errorf("skor %d seharusnya sah: %v", s, err)
		}
	}
	for _, s := range []int{-1, 101} {
		if err := CheckQuizScore(s); err == nil {
			t.Errorf("skor %d seharusnya ditolak", s)
		}
	}
}

func TestCountCompletedLessonsIgnoresUnknownAndDuplicate(t *testing.T) {
	completed := []string{"intro", "intro", "lama-dihapus", "fungsi"}
	if got := CountCompletedLessons(completed, sampleLessons); got != 2 {
		t.Fatalf("got %d, want 2", got)
	}
}

func TestIsModuleCompleted(t *testing.T) {
	if !IsModuleCompleted([]string{"error", "fungsi", "variabel", "intro"}, sampleLessons) {
		t.Error("semua lesson selesai harus completed")
	}
	if IsModuleCompleted([]string{"intro"}, sampleLessons) {
		t.Error("baru satu lesson belum completed")
	}
	if IsModuleCompleted(nil, nil) {
		t.Error("modul tanpa lesson tidak boleh dianggap completed")
	}
}

func TestSummarizeProgress(t *testing.T) {
	p := model.Progress{UserID: 1, ModuleID: 2, CompletedLessons: []string{"intro", "fungsi", "variabel"}}
	s := SummarizeProgress(p, sampleLessons)
	if s.TotalLessons != 4 || s.Percent != 75 || s.Completed {
		t.Fatalf("ringkasan salah: %+v", s)
	}
}
