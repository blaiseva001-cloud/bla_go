package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/argon2"
)

func HashPassword(pw string) string {
	salt := make([]byte, 16)
	_, _ = rand.Read(salt)
	key := argon2.IDKey([]byte(pw), salt, 1, 64*1024, 4, 32)
	return fmt.Sprintf("$argon2id$m=%d,t=%d,p=%d$%s$%s", 64*1024, 1, 4,
		base64.RawStdEncoding.EncodeToString(salt), base64.RawStdEncoding.EncodeToString(key))
}

func VerifyPassword(pw, encoded string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 5 {
		return false
	}
	var m, t, p int
	if _, err := fmt.Sscanf(parts[2], "m=%d,t=%d,p=%d", &m, &t, &p); err != nil {
		return false
	}
	salt, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false
	}
	want, err := base64.RawStdEncoding.DecodeString(parts[4])
	if err != nil {
		return false
	}
	got := argon2.IDKey([]byte(pw), salt, uint32(t), uint32(m), uint8(p), uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

func HashIP(ip string) string {
	h := sha256.Sum256([]byte(ip + "bla_salt_2026"))
	return hex.EncodeToString(h[:])
}

type Claims struct {
	UserID string `json:"uid"`
	Role   string `json:"rol"`
	jwt.RegisteredClaims
}

func SignToken(secret []byte, userID uuid.UUID, role string, expiry time.Duration) (string, error) {
	now := time.Now()
	return jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		UserID: userID.String(),
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			Issuer:    "bla.link",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}).SignedString(secret)
}

func ParseToken(secret []byte, raw string) (*Claims, error) {
	c := &Claims{}
	_, err := jwt.ParseWithClaims(raw, c, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("bad signing method")
		}
		return secret, nil
	})
	if err != nil {
		return nil, err
	}
	return c, nil
}
