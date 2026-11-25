package model

import "time"

type Lesson struct {
	Slug  string `bson:"slug" json:"slug" validate:"required"`
	Title string `bson:"title" json:"title" validate:"required"`
	Body  string `bson:"body" json:"body" validate:"required"`
}

type Chapter struct {
	Title   string   `bson:"title" json:"title" validate:"required"`
	Lessons []Lesson `bson:"lessons" json:"lessons" validate:"dive"`
}

type ModuleContent struct {
	ID        string    `bson:"_id,omitempty" json:"id"`
	ModuleID  int       `bson:"module_id" json:"module_id"`
	Title     string    `bson:"title" json:"title"`
	Chapters  []Chapter `bson:"chapters" json:"chapters"`
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}