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
)

var ErrContentNotFound = errors.New("konten modul belum dibuat")

type ContentRepository interface {
	EnsureIndexes(ctx context.Context) error
	GetByModuleID(ctx context.Context, moduleID int) (*model.ModuleContent, error)
	Upsert(ctx context.Context, content model.ModuleContent) (*model.ModuleContent, error)
}

type contentRepository struct {
	coll *mongo.Collection
}

func NewContentRepository(db *mongo.Database) ContentRepository {
	return &contentRepository{coll: db.Collection(CollectionModuleContents)}
}

func (r *contentRepository) EnsureIndexes(ctx context.Context) error {
	_, err := r.coll.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "module_id", Value: 1}},
		Options: options.Index().SetName("uq_module_content_module_id").SetUnique(true),
	})
	if err != nil {
		return fmt.Errorf("buat index konten modul: %w", err)
	}
	return nil
}

func (r *contentRepository) GetByModuleID(ctx context.Context, moduleID int) (*model.ModuleContent, error) {
	var mc model.ModuleContent
	err := r.coll.FindOne(ctx, bson.M{"module_id": moduleID}).Decode(&mc)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, ErrContentNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("baca konten modul: %w", err)
	}
	return &mc, nil
}

func (r *contentRepository) Upsert(ctx context.Context, content model.ModuleContent) (*model.ModuleContent, error) {
	now := time.Now().UTC()
	content.UpdatedAt = now

	update := bson.M{
		"$set": bson.M{
			"title":      content.Title,
			"chapters":   content.Chapters,
			"updated_at": content.UpdatedAt,
		},
		"$setOnInsert": bson.M{
			"module_id":  content.ModuleID,
			"created_at": now,
		},
	}

	opts := options.FindOneAndUpdate().
		SetUpsert(true).
		SetReturnDocument(options.After)

	var updated model.ModuleContent
	err := r.coll.FindOneAndUpdate(ctx, bson.M{"module_id": content.ModuleID}, update, opts).Decode(&updated)
	if err != nil {
		return nil, fmt.Errorf("simpan konten modul: %w", err)
	}
	return &updated, nil
}