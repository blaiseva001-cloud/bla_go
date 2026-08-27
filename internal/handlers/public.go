package handlers

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/blaiseva001-cloud/backend/internal/models"
	"github.com/blaiseva001-cloud/backend/internal/utils"
)

func (h *H) PublicSite(c fiber.Ctx) error {
	username := strings.ToLower(c.Params("username"))
	cacheKey := "pub:" + username
	if cached, err := h.KV.Get(c.Context(), cacheKey); err == nil {
		c.Set("Content-Type", "application/json")
		c.Set("X-Cache", "HIT")
		return c.SendString(cached)
	}
	var user models.User
	if err := h.DB.Where("username = ?", username).First(&user).Error; err != nil || !user.Verified || user.Banned {
		return fail(c, 404, "not found")
	}
	var site models.Site
	if err := h.DB.Preload("Links", func(db *gorm.DB) *gorm.DB { return db.Where("enabled = ?", true).Order("position asc") }).Where("user_id = ?", user.ID).First(&site).Error; err != nil {
		return fail(c, 404, "not found")
	}
	var theme any
	_ = json.Unmarshal([]byte(site.Theme), &theme)
	payload := fiber.Map{"username": user.Username, "display_name": site.DisplayName, "bio": site.Bio, "avatar_url": site.AvatarURL, "theme": theme, "links": site.Links}
	jsonBytes, _ := json.Marshal(payload)
	_ = h.KV.Set(c.Context(), cacheKey, string(jsonBytes), 60*time.Second)
	c.Set("X-Cache", "MISS")
	go func(sid uuid.UUID, ip, ref, ua string) {
		h.DB.Create(&models.PageView{SiteID: sid, IPHash: utils.HashIP(ip), Referrer: ref, UserAgent: ua})
	}(site.ID, c.IP(), string(c.Request().Header.Referer()), string(c.Request().Header.UserAgent()))
	return c.JSON(payload)
}

func (h *H) PublicClick(c fiber.Ctx) error {
	username := strings.ToLower(c.Params("username"))
	var user models.User
	if err := h.DB.Where("username = ?", username).First(&user).Error; err != nil {
		return fail(c, 404, "not found")
	}
	var site models.Site
	if err := h.DB.Where("user_id = ?", user.ID).First(&site).Error; err != nil {
		return fail(c, 404, "not found")
	}
	var link models.Link
	if err := h.DB.Where("id = ? AND site_id = ?", c.Params("link"), site.ID).First(&link).Error; err != nil {
		return fail(c, 404, "not found")
	}
	go func(sid, lid uuid.UUID) {
		h.DB.Create(&models.LinkClick{SiteID: sid, LinkID: lid})
		h.DB.Model(&models.Link{}).Where("id = ?", lid).UpdateColumn("clicks", gorm.Expr("clicks + 1"))
	}(site.ID, link.ID)
	return c.JSON(fiber.Map{"url": link.URL})
}
