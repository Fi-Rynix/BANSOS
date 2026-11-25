package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"

	"bansos-backend/app/model"
	"bansos-backend/database"
)

var ErrProgressNotFound = errors.New("progres tidak ditemukan")

type ProgressRepository interface {
	Get(ctx context.Context, userID, moduleID int) (*model.Progress, error)
	CompleteLesson(ctx context.Context, userID, moduleID int, lessonSlug string) (*model.Progress, error)
	SaveQuizScore(ctx context.Context, userID, moduleID int, quizSlug string, score int) (*model.Progress, error)
	EnsureIndexes(ctx context.Context) error
}

type progressRepository struct {
	coll *mongo.Collection
}

func NewProgressRepository(db *mongo.Database) ProgressRepository {
	return &progressRepository{coll: db.Collection(database.CollectionProgress)}
}

func progressFilter(userID, moduleID int) bson.D {
	return bson.D{{Key: "user_id", Value: userID}, {Key: "module_id", Value: moduleID}}
}

func (r *progressRepository) Get(ctx context.Context, userID, moduleID int) (*model.Progress, error) {
	var p model.Progress
	err := r.coll.FindOne(ctx, progressFilter(userID, moduleID)).Decode(&p)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrProgressNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("baca progres: %w", err)
	}
	return &p, nil
}

func (r *progressRepository) CompleteLesson(ctx context.Context, userID, moduleID int, lessonSlug string) (*model.Progress, error) {
	update := bson.D{
		{Key: "$addToSet", Value: bson.D{{Key: "completed_lessons", Value: lessonSlug}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "quiz_scores", Value: bson.D{}}}},
	}
	return r.upsert(ctx, userID, moduleID, update)
}

func (r *progressRepository) SaveQuizScore(ctx context.Context, userID, moduleID int, quizSlug string, score int) (*model.Progress, error) {
	update := bson.D{
		{Key: "$max", Value: bson.D{{Key: "quiz_scores." + quizSlug, Value: score}}},
		{Key: "$set", Value: bson.D{{Key: "updated_at", Value: time.Now().UTC()}}},
		{Key: "$setOnInsert", Value: bson.D{{Key: "completed_lessons", Value: bson.A{}}}},
	}
	return r.upsert(ctx, userID, moduleID, update)
}

func (r *progressRepository) upsert(ctx context.Context, userID, moduleID int, update bson.D) (*model.Progress, error) {
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var p model.Progress
	err := r.coll.FindOneAndUpdate(ctx, progressFilter(userID, moduleID), update, opts).Decode(&p)
	if mongo.IsDuplicateKeyError(err) {
		err = r.coll.FindOneAndUpdate(ctx, progressFilter(userID, moduleID), update, opts).Decode(&p)
	}
	if err != nil {
		return nil, fmt.Errorf("simpan progres: %w", err)
	}
	return &p, nil
}

func (r *progressRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "user_id", Value: 1}, {Key: "module_id", Value: 1}},
		Options: options.Index().SetName("uq_progress_user_module").SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("buat index progres: %w", err)
	}
	return nil
}
