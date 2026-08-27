package handlers

import (
	"errors"
	"log"
	"regexp"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/blaiseva001-cloud/backend/internal/models"
	"github.com/blaiseva001-cloud/backend/internal/services"
	"github.com/blaiseva001-cloud/backend/internal/utils"
)

var usernameRe = regexp.MustCompile(`^[a-z0-9_][a-z0-9_.-]{2,29}$`)

func norm(s string) string { return strings.ToLower(strings.TrimSpace(s)) }

func (h *H) issueAndSend(c fiber.Ctx, email string) (int64, error) {
	code, exp, err := h.OTP.Issue(c.Context(), email)
	if err != nil {
		return 0, err
	}
	log.Printf("[otp] %s -> %s", email, code)
	go func() { _ = services.SendOTPEmail(h.Cfg, email, code, int(h.Cfg.OTPExpiry.Minutes())) }()
	return exp, nil
}

func (h *H) CheckEmail(c fiber.Ctx) error {
	var b struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := bind(c, &b); err != nil || validate.Struct(b) != nil {
		return fail(c, 422, "invalid")
	}
	var n int64
	h.DB.Model(&models.User{}).Where("email = ?", norm(b.Email)).Count(&n)
	return c.JSON(fiber.Map{"exists": n > 0, "available": n == 0})
}

func (h *H) CheckUsername(c fiber.Ctx) error {
	u := norm(c.Params("username"))
	v := usernameRe.MatchString(u)
	var n int64
	if v {
		h.DB.Model(&models.User{}).Where("username = ?", u).Count(&n)
	}
	return c.JSON(fiber.Map{"valid": v, "available": v && n == 0})
}

func (h *H) Register(c fiber.Ctx) error {
	var b struct {
		Email    string `json:"email" validate:"required,email"`
		Username string `json:"username" validate:"required,min=3,max=30"`
		Password string `json:"password" validate:"required,min=8"`
	}
	if err := bind(c, &b); err != nil || validate.Struct(b) != nil {
		return fail(c, 422, "invalid")
	}
	b.Email, b.Username = norm(b.Email), norm(b.Username)
	if !usernameRe.MatchString(b.Username) {
		return fail(c, 422, "invalid username")
	}
	var n int64
	h.DB.Model(&models.User{}).Where("email = ?", b.Email).Count(&n)
	if n > 0 {
		return fail(c, 409, "email exists")
	}
	h.DB.Model(&models.User{}).Where("username = ?", b.Username).Count(&n)
	if n > 0 {
		return fail(c, 409, "username taken")
	}
	user := models.User{Email: b.Email, Username: b.Username, Password: utils.HashPassword(b.Password), Role: "user"}
	if err := h.DB.Create(&user).Error; err != nil {
		return fail(c, 500, "db error")
	}
	exp, err := h.issueAndSend(c, b.Email)
	if err != nil {
		if errors.Is(err, services.ErrCooldown) {
			return c.Status(429).JSON(fiber.Map{"error": "cooldown", "retry_in": h.OTP.CooldownLeft(c.Context(), b.Email)})
		}
		return fail(c, 500, "otp error")
	}
	return c.Status(201).JSON(fiber.Map{"message": "otp sent", "expires_in": exp, "user": user})
}

func (h *H) ensureSite(u *models.User) {
	var n int64
	h.DB.Model(&models.Site{}).Where("user_id = ?", u.ID).Count(&n)
	if n == 0 {
		h.DB.Create(&models.Site{UserID: u.ID, DisplayName: u.Username, Theme: `{"accent":"green"}`})
	}
}

func (h *H) Verify(c fiber.Ctx) error {
	var b struct {
		Email string `json:"email" validate:"required,email"`
		Code  string `json:"code" validate:"required,len=6"`
	}
	if err := bind(c, &b); err != nil || validate.Struct(b) != nil {
		return fail(c, 422, "invalid")
	}
	email := norm(b.Email)
	var user models.User
	if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return fail(c, 404, "no account")
	}
	if user.Banned {
		return fail(c, 403, "banned")
	}
	if !h.OTP.Verify(c.Context(), email, b.Code) {
		return fail(c, 400, "bad code")
	}
	h.DB.Model(&user).Update("verified", true)
	user.Verified = true
	h.ensureSite(&user)
	token, _ := utils.SignToken(h.Cfg.JWTSecret, user.ID, user.Role, h.Cfg.JWTExpiry)
	return c.JSON(fiber.Map{"token": token, "user": user})
}

func (h *H) Resend(c fiber.Ctx) error {
	var b struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := bind(c, &b); err != nil || validate.Struct(b) != nil {
		return fail(c, 422, "invalid")
	}
	email := norm(b.Email)
	var user models.User
	if err := h.DB.Where("email = ?", email).First(&user).Error; err != nil {
		return fail(c, 404, "no account")
	}
	if user.Verified {
		return fail(c, 400, "verified")
	}
	exp, err := h.issueAndSend(c, user.Email)
	if err != nil {
		if errors.Is(err, services.ErrCooldown) {
			return c.Status(429).JSON(fiber.Map{"error": "cooldown", "retry_in": h.OTP.CooldownLeft(c.Context(), email)})
		}
		return fail(c, 500, "error")
	}
	return c.JSON(fiber.Map{"message": "sent", "expires_in": exp})
}

func (h *H) Login(c fiber.Ctx) error {
	var b struct {
		Email    string `json:"email" validate:"required,email"`
		Password string `json:"password" validate:"required"`
	}
	if err := bind(c, &b); err != nil || validate.Struct(b) != nil {
		return fail(c, 422, "invalid")
	}
	var user models.User
	if err := h.DB.Where("email = ?", norm(b.Email)).First(&user).Error; err != nil {
		return fail(c, 401, "bad creds")
	}
	if !user.Verified {
		return c.Status(403).JSON(fiber.Map{"error": "unverified", "code": "unverified"})
	}
	if user.Banned {
		return fail(c, 403, "banned")
	}
	if !utils.VerifyPassword(b.Password, user.Password) {
		return fail(c, 401, "bad creds")
	}
	token, _ := utils.SignToken(h.Cfg.JWTSecret, user.ID, user.Role, h.Cfg.JWTExpiry)
	return c.JSON(fiber.Map{"token": token, "user": user})
}

func (h *H) Me(c fiber.Ctx) error {
	var user models.User
	if err := h.DB.First(&user, "id = ?", userID(c)).Error; err != nil {
		return fail(c, 404, "gone")
	}
	return c.JSON(fiber.Map{"user": user})
}

func (h *H) Logout(c fiber.Ctx) error {
	jti := c.Locals("token_jti").(string)
	exp := c.Locals("token_exp").(int64)
	if ttl := time.Until(time.Unix(exp, 0)); ttl > 0 {
		_ = h.KV.Set(c.Context(), "blacklist:"+jti, "1", ttl)
	}
	return c.JSON(fiber.Map{"message": "out"})
}
