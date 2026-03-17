// Package analytics provides a Google Analytics plugin for goblog.
// It injects the GA tracking script into the <head> of every page
// when enabled and configured with a valid measurement ID.
package analytics

import (
	"goblog/plugin"
	"regexp"

	"gorm.io/gorm"
)

var validTrackingID = regexp.MustCompile(`^[A-Za-z0-9-]+$`)

// AnalyticsPlugin implements Google Analytics tracking.
type AnalyticsPlugin struct {
	plugin.BasePlugin
}

// New creates a new analytics plugin.
func New() *AnalyticsPlugin {
	return &AnalyticsPlugin{}
}

func (p *AnalyticsPlugin) Name() string        { return "analytics" }
func (p *AnalyticsPlugin) DisplayName() string { return "Google Analytics" }
func (p *AnalyticsPlugin) Version() string     { return "1.0.0" }

func (p *AnalyticsPlugin) Settings() []plugin.SettingDefinition {
	return []plugin.SettingDefinition{
		{
			Key:          "tracking_id",
			Type:         "text",
			DefaultValue: "",
			Label:        "Measurement ID",
			Description:  "Google Analytics measurement ID (e.g. G-XXXXXXXXXX)",
		},
		{
			Key:          "enabled",
			Type:         "text",
			DefaultValue: "false",
			Label:        "Enabled",
			Description:  "Set to 'true' to enable tracking",
		},
	}
}

func (p *AnalyticsPlugin) OnInit(db *gorm.DB) error { return nil }

func (p *AnalyticsPlugin) TemplateHead(ctx *plugin.HookContext) string {
	trackingID := ctx.Settings["tracking_id"]
	enabled := ctx.Settings["enabled"]
	if trackingID == "" || enabled != "true" {
		return ""
	}
	if !validTrackingID.MatchString(trackingID) {
		return ""
	}
	return `<script async src="https://www.googletagmanager.com/gtag/js?id=` + trackingID + `"></script>
<script>
  window.dataLayer = window.dataLayer || [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());
  gtag('config', '` + trackingID + `');
</script>`
}
