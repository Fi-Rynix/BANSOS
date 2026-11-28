package service

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5/pgxpool"

	"bansos-backend/app/model"
	"bansos-backend/app/repository"
	"bansos-backend/helper"
)

type ContentService struct {
	pool       *pgxpool.Pool
	contentRepo repository.ContentRepository
	moduleRepo  repository.ModuleRepository
}

func NewContentService(pool *pgxpool.Pool, contentRepo repository.ContentRepository, moduleRepo repository.ModuleRepository) *ContentService {
	return &ContentService{
		pool:        pool,
		contentRepo: contentRepo,
		moduleRepo:  moduleRepo,
	}
}

func (s *ContentService) Get(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id modul tidak valid")
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	if _, err := s.moduleRepo.FindByID(ctx, s.pool, id); err != nil {
		return helper.NotFound("modul tidak ditemukan")
	}

	content, err := s.contentRepo.GetByModuleID(ctx, id)
	if errors.Is(err, repository.ErrContentNotFound) {
		return helper.NotFound("konten modul belum tersedia")
	}
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "OK", "konten modul", content)
}

type UpsertContentRequest struct {
	Title    string          `json:"title" validate:"required,min=3,max=150"`
	Chapters []model.Chapter `json:"chapters" validate:"required,min=1,dive"`
}

func (s *ContentService) Upsert(c *fiber.Ctx) error {
	id, ok := helper.ParamID(c)
	if !ok {
		return helper.BadRequest("id modul tidak valid")
	}

	var req UpsertContentRequest
	if err := c.BodyParser(&req); err != nil {
		return helper.BadRequest("body JSON tidak valid")
	}

	req.Title = strings.TrimSpace(req.Title)
	if fields := helper.ValidateStruct(req); fields != nil {
		return helper.Validation(fields)
	}

	ctx, cancel := helper.RequestContext(c)
	defer cancel()

	if _, err := s.moduleRepo.FindByID(ctx, s.pool, id); err != nil {
		return helper.NotFound("modul tidak ditemukan")
	}

	content := model.ModuleContent{
		ModuleID: id,
		Title:    req.Title,
		Chapters: req.Chapters,
	}

	saved, err := s.contentRepo.Upsert(ctx, content)
	if err != nil {
		return helper.Internal(err)
	}

	return helper.Success(c, fiber.StatusOK, "OK", "konten modul berhasil disimpan", saved)
}