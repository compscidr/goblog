package blog

import "time"

// Page types
const (
	PageTypeWriting  = "writing"
	PageTypeResearch = "research"
	PageTypeAbout    = "about"
	PageTypeCustom   = "custom"
)

// Page represents a configurable site page (nav items, content pages, etc.)
type Page struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `sql:"index" json:"deleted_at,omitempty"`
	Title     string     `json:"title"`
	Slug      string     `gorm:"uniqueIndex" json:"slug"`
	Content   string     `sql:"type:text;" json:"content"`
	HeroURL   string     `json:"hero_url"`
	HeroType  string     `json:"hero_type"` // "image" or "video"
	PageType  string     `json:"page_type"` // "writing", "research", "about", "custom"
	ShowInNav bool       `json:"show_in_nav"`
	NavOrder  int        `json:"nav_order"`
	Enabled   bool       `json:"enabled"`
	ScholarID string     `json:"scholar_id,omitempty"`
}

// PagePermalink returns the URL path for this page
func (p Page) PagePermalink() string {
	return "/" + p.Slug
}

// IsVideo returns true if the hero media is a video
func (p Page) IsVideo() bool {
	return p.HeroType == "video"
}

// HasHero returns true if a hero URL is configured
func (p Page) HasHero() bool {
	return p.HeroURL != ""
}
