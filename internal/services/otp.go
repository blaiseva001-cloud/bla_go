package services

import (
	"context"
	"crypto/rand"
	"errors"
	"time"

	"github.com/blaiseva001-cloud/backend/internal/kv"
)

var ErrCooldown = errors.New("cooldown active")

type OTP struct {
	store    kv.KV
	Expiry   time.Duration
	Cooldown time.Duration
}

func NewOTP(store kv.KV, expiry, cooldown time.Duration) *OTP {
	return &OTP{store: store, Expiry: expiry, Cooldown: cooldown}
}

func (o *OTP) Issue(ctx context.Context, email string) (string, int64, error) {
	cd := "otp:cd:" + email
	if ttl, err := o.store.TTL(ctx, cd); err == nil && ttl > 0 {
		return "", int64(ttl.Seconds()), ErrCooldown
	}
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	code := make([]byte, 6)
	for i, c := range b {
		code[i] = byte('0' + int(c)%10)
	}
	_ = o.store.Set(ctx, "otp:"+email, string(code), o.Expiry)
	_ = o.store.Set(ctx, cd, "1", o.Cooldown)
	return string(code), int64(o.Expiry.Seconds()), nil
}

func (o *OTP) Verify(ctx context.Context, email, code string) bool {
	v, err := o.store.Get(ctx, "otp:"+email)
	if err != nil || v != code {
		return false
	}
	_ = o.store.Del(ctx, "otp:"+email)
	return true
}

func (o *OTP) CooldownLeft(ctx context.Context, email string) int64 {
	ttl, err := o.store.TTL(ctx, "otp:cd:"+email)
	if err != nil || ttl < 0 {
		return 0
	}
	return int64(ttl.Seconds())
}
