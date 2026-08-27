package handlers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/blaiseva001-cloud/backend/internal/models"
)

func (h *H) bust(uid uuid.UUID) {
	var u models.User
	if err := h.DB.First(&u, "id = ?", uid).Error; err == nil {
		_ = h.KV.Del(context.Background(), "pub:"+u.Username)
	}
}

func (h *H) mySite(c fiber.Ctx) (*models.Site, error) {
	var site models.Site
	err := h.DB.Preload("Links", func(db *gorm.DB) *gorm.DB { return db.Order("position asc") }).
		Where("user_id = ?", userID(c)).First(&site).Error
	if err != nil {
		var user models.User
		if e := h.DB.First(&user, "id = ?", userID(c)).Error; e != nil {
			return nil, e
		}
		site = models.Site{UserID: user.ID, DisplayName: user.Username, Theme: `{"accent":"green"}`}
		if e := h.DB.Create(&site).Error; e != nil {
			return nil, e
		}
	}
	return &site, nil
}

func (h *H) GetSite(c fiber.Ctx) error {
	s, err := h.mySite(c)
	if err != nil {
		return fail(c, 500, "err")
	}
	return c.JSON(fiber.Map{"site": s})
}

func (h *H) UpdateSite(c fiber.Ctx) error {
	s, err := h.mySite(c)
	if err != nil {
		return fail(c, 500, "err")
	}
	var b struct {
		DisplayName *string        `json:"display_name"`
		Bio         *string        `json:"bio"`
		AvatarURL   *string        `json:"avatar_url"`
		Theme       map[string]any `json:"theme"`
	}
	if err := bind(c, &b); err != nil {
		return fail(c, 422, "invalid")
	}
	if b.DisplayName != nil {
		s.DisplayName = *b.DisplayName
	}
	if b.Bio != nil {
		s.Bio = *b.Bio
	}
	if b.AvatarURL != nil {
		s.AvatarURL = *b.AvatarURL
	}
	if b.Theme != nil {
		raw, _ := json.Marshal(b.Theme)
		s.Theme = string(raw)
	}
	h.DB.Save(s)
	h.bust(userID(c))
	return c.JSON(fiber.Map{"site": s})
}

func (h *H) AddLink(c fiber.Ctx) error {
	s, err := h.mySite(c)
	if err != nil {
		return fail(c, 500, "err")
	}
	var b struct {
		Title string `json:"title" validate:"required,max=120"`
		URL   string `json:"url" validate:"required,url,max=2000"`
	}
	if err := bind(c, &b); err != nil || validate.Struct(b) != nil {
		return fail(c, 422, "title + valid url required")
	}
	var maxPos int
	h.DB.Model(&models.Link{}).Where("site_id = ?", s.ID).Select("COALESCE(MAX(position),0)").Scan(&maxPos)
	l := models.Link{SiteID: s.ID, Title: b.Title, URL: b.URL, Position: maxPos + 1, Enabled: true}
	h.DB.Create(&l)
	h.bust(userID(c))
	return c.Status(201).JSON(fiber.Map{"link": l})
}

func (h *H) UpdateLink(c fiber.Ctx) error {
	s, err := h.mySite(c)
	if err != nil {
		return fail(c, 500, "err")
	}
	var l models.Link
	if err := h.DB.Where("id = ? AND site_id = ?", c.Params("id"), s.ID).First(&l).Error; err != nil {
		return fail(c, 404, "not found")
	}
	var b struct {
		Title   *string `json:"title"`
		URL     *string `json:"url"`
		Enabled *bool   `json:"enabled"`
	}
	if err := bind(c, &b); err != nil {
		return fail(c, 422, "invalid")
	}
	if b.Title != nil {
		l.Title = *b.Title
	}
	if b.URL != nil {
		l.URL = *b.URL
	}
	if b.Enabled != nil {
		l.Enabled = *b.Enabled
	}
	h.DB.Save(&l)
	h.bust(userID(c))
	return c.JSON(fiber.Map{"link": l})
}

func (h *H) DeleteLink(c fiber.Ctx) error {
	s, err := h.mySite(c)
	if err != nil {
		return fail(c, 500, "err")
	}
	if h.DB.Where("id = ? AND site_id = ?", c.Params("id"), s.ID).Delete(&models.Link{}).RowsAffected == 0 {
		return fail(c, 404, "not found")
	}
	h.bust(userID(c))
	return c.JSON(fiber.Map{"message": "ok"})
}

func (h *H) ReorderLinks(c fiber.Ctx) error {
	s, err := h.mySite(c)
	if err != nil {
		return fail(c, 500, "err")
	}
	var b struct {
		IDs []string `json:"ids"`
	}
	if err := bind(c, &b); err != nil {
		return fail(c, 422, "invalid")
	}
	tx := h.DB.Begin()
	for i, id := range b.IDs {
		tx.Model(&models.Link{}).Where("id = ? AND site_id = ?", id, s.ID).Update("position", i+1)
	}
	tx.Commit()
	h.bust(userID(c))
	return c.JSON(fiber.Map{"message": "ok"})
}

func (h *H) MyAnalytics(c fiber.Ctx) error {
	s, err := h.mySite(c)
	if err != nil {
		return fail(c, 500, "err")
	}
	var v, cl int64
	h.DB.Model(&models.PageView{}).Where("site_id = ?", s.ID).Count(&v)
	h.DB.Model(&models.LinkClick{}).Where("site_id = ?", s.ID).Count(&cl)
	series := make([]fiber.Map, 0, 7)
	day := time.Now().UTC().Truncate(24 * time.Hour)
	for i := 6; i >= 0; i-- {
		start := day.Add(-time.Duration(i) * 24 * time.Hour)
		end := start.Add(24 * time.Hour)
		var sv, sc int64
		h.DB.Model(&models.PageView{}).Where("site_id = ? AND created_at >= ? AND created_at < ?", s.ID, start, end).Count(&sv)
		h.DB.Model(&models.LinkClick{}).Where("site_id = ? AND created_at >= ? AND created_at < ?", s.ID, start, end).Count(&sc)
		series = append(series, fiber.Map{"date": start.Format("01-02"), "views": sv, "clicks": sc})
	}
	return c.JSON(fiber.Map{"views": v, "clicks": cl, "series": series})
}
