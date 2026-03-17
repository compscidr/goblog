// Package socialicons provides a plugin that renders social media icon links.
// When enabled, it injects social icon HTML into the template footer,
// replacing the need for hardcoded social URLs in theme templates.
package socialicons

import (
	"goblog/plugin"
	"html"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type social struct {
	key   string // setting key
	label string // display label
	icon  string // Font Awesome icon class
}

var socials = []social{
	{"github_url", "GitHub", "fab fa-github"},
	{"linkedin_url", "LinkedIn", "fab fa-linkedin"},
	{"x_url", "X", "fab fa-x"},
	{"keybase_url", "Keybase", "fab fa-keybase"},
	{"instagram_url", "Instagram", "fab fa-instagram"},
	{"facebook_url", "Facebook", "fab fa-facebook"},
	{"strava_url", "Strava", "fab fa-strava"},
	{"spotify_url", "Spotify", "fab fa-spotify"},
	{"xbox_url", "Xbox", "fab fa-xbox"},
	{"steam_url", "Steam", "fab fa-steam"},
}

// SocialIconsPlugin renders social media icon links in the footer.
type SocialIconsPlugin struct {
	plugin.BasePlugin
}

// New creates a new social icons plugin.
func New() *SocialIconsPlugin {
	return &SocialIconsPlugin{}
}

func (p *SocialIconsPlugin) Name() string        { return "socialicons" }
func (p *SocialIconsPlugin) DisplayName() string { return "Social Icons" }
func (p *SocialIconsPlugin) Version() string     { return "1.0.0" }

func (p *SocialIconsPlugin) OnInit(db *gorm.DB) error { return nil }

func (p *SocialIconsPlugin) Settings() []plugin.SettingDefinition {
	defs := []plugin.SettingDefinition{
		{Key: "enabled", Type: "text", DefaultValue: "true", Label: "Enabled", Description: "Set to 'true' to show social icons"},
	}
	for _, s := range socials {
		defs = append(defs, plugin.SettingDefinition{
			Key:          s.key,
			Type:         "text",
			DefaultValue: "",
			Label:        s.label + " URL",
			Description:  "Full URL to your " + s.label + " profile",
		})
	}
	return defs
}

func (p *SocialIconsPlugin) TemplateData(ctx *plugin.HookContext) gin.H {
	if ctx.Settings["enabled"] != "true" {
		return nil
	}
	// Build a list of active social links for templates that want structured data
	type socialLink struct {
		Name string
		URL  string
		Icon string
	}
	var links []socialLink
	for _, s := range socials {
		if url := ctx.Settings[s.key]; url != "" {
			links = append(links, socialLink{Name: s.label, URL: url, Icon: s.icon})
		}
	}
	return gin.H{"links": links}
}

func (p *SocialIconsPlugin) TemplateFooter(ctx *plugin.HookContext) string {
	if ctx.Settings["enabled"] != "true" {
		return ""
	}
	out := `<div class="text-center" style="padding: 10px 0;">`
	for _, s := range socials {
		url := ctx.Settings[s.key]
		if url == "" {
			continue
		}
		safeURL := html.EscapeString(url)
		safeLabel := html.EscapeString(s.label)
		out += `<a href="` + safeURL + `" target="_blank" rel="noopener noreferrer" title="` + safeLabel + `" style="margin: 0 6px; color: inherit;"><i class="` + s.icon + ` fa-1x"></i></a>`
	}
	out += `</div>`
	return out
}
