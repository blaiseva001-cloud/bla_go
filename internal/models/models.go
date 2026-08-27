package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	Email     string    `gorm:"size:255;uniqueIndex;not null" json:"email"`
	Username  string    `gorm:"size:30;uniqueIndex;not null" json:"username"`
	Password  string    `gorm:"size:255;not null" json:"-"`
	Role      string    `gorm:"size:10;not null;default:user" json:"role"`
	Verified  bool      `gorm:"not null;default:false" json:"verified"`
	Banned    bool      `gorm:"not null;default:false" json:"banned"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Site struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	UserID      uuid.UUID `gorm:"type:uuid;uniqueIndex;not null" json:"user_id"`
	DisplayName string    `gorm:"size:80" json:"display_name"`
	Bio         string    `gorm:"size:500" json:"bio"`
	AvatarURL   string    `gorm:"size:2000" json:"avatar_url"`
	Theme       string    `gorm:"type:text;not null;default:'{\"accent\":\"green\"}'" json:"theme"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Links       []Link    `gorm:"foreignKey:SiteID" json:"links,omitempty"`
	User        *User     `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

type Link struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SiteID    uuid.UUID `gorm:"type:uuid;index;not null" json:"site_id"`
	Title     string    `gorm:"size:120;not null" json:"title"`
	URL       string    `gorm:"size:2000;not null" json:"url"`
	Position  int       `gorm:"not null;default:0" json:"position"`
	Enabled   bool      `gorm:"not null;default:true" json:"enabled"`
	Clicks    int       `gorm:"not null;default:0" json:"clicks"`
	CreatedAt time.Time `json:"created_at"`
}

type PageView struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SiteID    uuid.UUID `gorm:"type:uuid;index;not null" json:"site_id"`
	IPHash    string    `gorm:"size:64;index" json:"-"`
	Referrer  string    `gorm:"size:2000" json:"referrer"`
	UserAgent string    `gorm:"size:500" json:"user_agent"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

type LinkClick struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey" json:"id"`
	SiteID    uuid.UUID `gorm:"type:uuid;index;not null" json:"site_id"`
	LinkID    uuid.UUID `gorm:"type:uuid;index;not null" json:"link_id"`
	CreatedAt time.Time `gorm:"index" json:"created_at"`
}

func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	if u.Role == "" {
		u.Role = "user"
	}
	return nil
}
func (s *Site) BeforeCreate(tx *gorm.DB) error      { if s.ID == uuid.Nil { s.ID = uuid.New() }; return nil }
func (l *Link) BeforeCreate(tx *gorm.DB) error      { if l.ID == uuid.Nil { l.ID = uuid.New() }; return nil }
func (p *PageView) BeforeCreate(tx *gorm.DB) error  { if p.ID == uuid.Nil { p.ID = uuid.New() }; return nil }
func (l *LinkClick) BeforeCreate(tx *gorm.DB) error { if l.ID == uuid.Nil { l.ID = uuid.New() }; return nil }
