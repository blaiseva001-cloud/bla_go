package handlers

import (
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/blaiseva001-cloud/backend/internal/models"
)

func (h *H) AdminStats(c fiber.Ctx) error {
	var u, v, b, s, l, pv, cl int64
	h.DB.Model(&models.User{}).Count(&u)
	h.DB.Model(&models.User{}).Where("verified = ?", true).Count(&v)
	h.DB.Model(&models.User{}).Where("banned = ?", true).Count(&b)
	h.DB.Model(&models.Site{}).Count(&s)
	h.DB.Model(&models.Link{}).Count(&l)
	h.DB.Model(&models.PageView{}).Count(&pv)
	h.DB.Model(&models.LinkClick{}).Count(&cl)
	signups := make([]fiber.Map, 0, 7)
	day := time.Now().UTC().Truncate(24 * time.Hour)
	for i := 6; i >= 0; i-- {
		start := day.Add(-time.Duration(i) * 24 * time.Hour)
		end := start.Add(24 * time.Hour)
		var n int64
		h.DB.Model(&models.User{}).Where("created_at >= ? AND created_at < ?", start, end).Count(&n)
		signups = append(signups, fiber.Map{"date": start.Format("01-02"), "count": n})
	}
	return c.JSON(fiber.Map{"users": u, "verified": v, "banned": b, "sites": s, "links": l, "views": pv, "clicks": cl, "signups": signups})
}

func (h *H) AdminUsers(c fiber.Ctx) error {
	q := strings.ToLower(c.Query("q"))
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}
	tx := h.DB.Model(&models.User{})
	if q != "" {
		tx = tx.Where("email ILIKE ? OR username ILIKE ?", "%"+q+"%", "%"+q+"%")
	}
	var total int64
	tx.Count(&total)
	var users []models.User
	tx.Order("created_at desc").Limit(20).Offset((page - 1) * 20).Find(&users)
	return c.JSON(fiber.Map{"users": users, "total": total, "page": page})
}

func (h *H) AdminSetRole(c fiber.Ctx) error {
	var b struct {
		Role string `json:"role"`
	}
	if err := bind(c, &b); err != nil || (b.Role != "user" && b.Role != "admin") {
		return fail(c, 422, "invalid")
	}
	if c.Params("id") == c.Locals("user_id").(string) && b.Role != "admin" {
		return fail(c, 400, "cannot demote self")
	}
	if h.DB.Model(&models.User{}).Where("id = ?", c.Params("id")).Update("role", b.Role).RowsAffected == 0 {
		return fail(c, 404, "not found")
	}
	return c.JSON(fiber.Map{"message": "ok"})
}

func (h *H) AdminSetBan(c fiber.Ctx) error {
	var b struct {
		Banned bool `json:"banned"`
	}
	if err := bind(c, &b); err != nil {
		return fail(c, 422, "invalid")
	}
	if c.Params("id") == c.Locals("user_id").(string) {
		return fail(c, 400, "cannot ban self")
	}
	if h.DB.Model(&models.User{}).Where("id = ?", c.Params("id")).Update("banned", b.Banned).RowsAffected == 0 {
		return fail(c, 404, "not found")
	}
	return c.JSON(fiber.Map{"message": "ok"})
}

func (h *H) AdminDeleteUser(c fiber.Ctx) error {
	target := c.Params("id")
	if target == c.Locals("user_id").(string) {
		return fail(c, 400, "cannot delete self")
	}
	tx := h.DB.Begin()
	var sids []uuid.UUID
	tx.Model(&models.Site{}).Where("user_id = ?", target).Pluck("id", &sids)
	if len(sids) > 0 {
		tx.Where("site_id IN ?", sids).Delete(&models.PageView{})
		tx.Where("site_id IN ?", sids).Delete(&models.LinkClick{})
		tx.Where("site_id IN ?", sids).Delete(&models.Link{})
		tx.Where("user_id = ?", target).Delete(&models.Site{})
	}
	tx.Where("id = ?", target).Delete(&models.User{})
	tx.Commit()
	return c.JSON(fiber.Map{"message": "ok"})
}

func (h *H) AdminSites(c fiber.Ctx) error {
	var sites []models.Site
	h.DB.Preload("User").Order("created_at desc").Limit(50).Find(&sites)
	return c.JSON(fiber.Map{"sites": sites})
}

func (h *H) AdminDeleteSite(c fiber.Ctx) error {
	id := c.Params("id")
	tx := h.DB.Begin()
	tx.Where("site_id = ?", id).Delete(&models.PageView{})
	tx.Where("site_id = ?", id).Delete(&models.LinkClick{})
	tx.Where("site_id = ?", id).Delete(&models.Link{})
	tx.Where("id = ?", id).Delete(&models.Site{})
	tx.Commit()
	return c.JSON(fiber.Map{"message": "ok"})
}
