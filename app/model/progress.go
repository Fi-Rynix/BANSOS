package model

import "time"

type Progress struct {
	UserID           int            `bson:"user_id" json:"user_id"`
	ModuleID         int            `bson:"module_id" json:"module_id"`
	CompletedLessons []string       `bson:"completed_lessons" json:"completed_lessons"`
	QuizScores       map[string]int `bson:"quiz_scores" json:"quiz_scores"`
	UpdatedAt        time.Time      `bson:"updated_at" json:"updated_at"`
}

type CompleteLessonRequest struct {
	LessonSlug string `json:"lesson_slug" validate:"required,max=100,nospace"`
}

type SubmitQuizRequest struct {
	QuizSlug string         `json:"quiz_slug" validate:"required,max=100,nospace"`
	Answers  map[string]int `json:"answers" validate:"required,min=1"`
}

type ProgressSummary struct {
	Progress
	TotalLessons int  `json:"total_lessons"`
	Percent      int  `json:"percent"`
	Completed    bool `json:"completed"`
}
