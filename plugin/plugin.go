// Package plugin defines the interface for goblog plugins.
// Plugins can inject data into templates, add HTML to <head> and <body>,
// register scheduled background jobs, and define their own settings.
//
// Plugins can be compiled-in (imported and registered in main()) or
// loaded dynamically from .go files in the plugins/ directory at startup.
package plugin

import (
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SettingDefinition describes a single setting that a plugin requires.
// Settings are stored in the blog's Setting table namespaced as "pluginname.key".
type SettingDefinition struct {
	Key          string // short key, e.g. "tracking_id"
	Type         string // "text", "textarea", "file", "bool"
	DefaultValue string
	Label        string // human-readable label for admin UI
	Description  string // help text
}

// ScheduledJob describes a periodic task the plugin wants to run.
type ScheduledJob struct {
	Name     string
	Interval time.Duration
	Run      func(db *gorm.DB, settings map[string]string) error
}

// HookContext provides everything a plugin hook needs.
type HookContext struct {
	GinContext *gin.Context
	DB         *gorm.DB
	Settings   map[string]string // plugin's own settings (namespace prefix stripped)
	Template   string            // which template is being rendered
	Data       gin.H             // the existing template data (read-only)
}

// Plugin is the core interface. Embed BasePlugin to get no-op defaults
// and only implement the methods you need.
type Plugin interface {
	Name() string
	DisplayName() string
	Version() string
	Settings() []SettingDefinition
	ScheduledJobs() []ScheduledJob
	TemplateData(ctx *HookContext) gin.H
	TemplateHead(ctx *HookContext) string
	TemplateFooter(ctx *HookContext) string
	OnInit(db *gorm.DB) error
}

// BasePlugin provides no-op implementations of all Plugin methods.
type BasePlugin struct{}

func (BasePlugin) Settings() []SettingDefinition           { return nil }
func (BasePlugin) ScheduledJobs() []ScheduledJob           { return nil }
func (BasePlugin) TemplateData(ctx *HookContext) gin.H     { return nil }
func (BasePlugin) TemplateHead(ctx *HookContext) string    { return "" }
func (BasePlugin) TemplateFooter(ctx *HookContext) string  { return "" }
func (BasePlugin) OnInit(db *gorm.DB) error                { return nil }
