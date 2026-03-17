// Package scholar provides a Google Scholar integration plugin for goblog.
// It displays academic publications on a dynamic "research" page, with
// caching and throttle resilience via the compscidr/scholar library.
package scholar

import (
	"fmt"
	"goblog/blog"
	"log"
	"sort"
	"time"

	gplugin "goblog/plugin"

	scholarlib "github.com/compscidr/scholar"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ScholarPlugin displays Google Scholar publications.
type ScholarPlugin struct {
	gplugin.BasePlugin
	sch *scholarlib.Scholar
}

// New creates a new scholar plugin.
func New() *ScholarPlugin {
	return &ScholarPlugin{}
}

func (p *ScholarPlugin) Name() string        { return "scholar" }
func (p *ScholarPlugin) DisplayName() string { return "Google Scholar" }
func (p *ScholarPlugin) Version() string     { return "1.0.0" }

func (p *ScholarPlugin) Settings() []gplugin.SettingDefinition {
	return []gplugin.SettingDefinition{
		{Key: "enabled", Type: "text", DefaultValue: "false", Label: "Enabled", Description: "Set to 'true' to enable the research page"},
		{Key: "scholar_id", Type: "text", DefaultValue: "", Label: "Google Scholar ID", Description: "Your Google Scholar profile ID (e.g. SbUmSEAAAAAJ)"},
		{Key: "article_limit", Type: "text", DefaultValue: "50", Label: "Article Limit", Description: "Maximum number of articles to display"},
		{Key: "profile_cache", Type: "text", DefaultValue: "profiles.json", Label: "Profile Cache File", Description: "File path for profile cache"},
		{Key: "article_cache", Type: "text", DefaultValue: "articles.json", Label: "Article Cache File", Description: "File path for article cache"},
	}
}

func (p *ScholarPlugin) OnInit(db *gorm.DB) error {
	// Ensure a research page exists in the pages table.
	// The user can customize title, slug, hero, nav order via admin.
	var page blog.Page
	result := db.Where("page_type = ?", "research").First(&page)
	if result.Error != nil {
		// No research page exists — create the default
		page = blog.Page{
			Title:    "Research",
			Slug:     "research",
			PageType: "research",
			ShowInNav: true,
			NavOrder: 20,
			Enabled:  true,
		}
		db.Create(&page)
		log.Println("Scholar plugin: created research page")
	}

	// Migrate ScholarID from page record to plugin settings (backward compat)
	if page.ScholarID != "" {
		var existing gplugin.PluginSetting
		if err := db.Where("plugin_name = ? AND key = ?", "scholar", "scholar_id").First(&existing).Error; err != nil || existing.Value == "" {
			db.Where("plugin_name = ? AND key = ?", "scholar", "scholar_id").
				Assign(gplugin.PluginSetting{Value: page.ScholarID}).
				FirstOrCreate(&gplugin.PluginSetting{PluginName: "scholar", Key: "scholar_id", Value: page.ScholarID})
			log.Printf("Scholar plugin: migrated scholar_id %s from page to plugin settings", page.ScholarID)
		}
	}

	return nil
}

func (p *ScholarPlugin) Pages() []gplugin.PageDefinition {
	return []gplugin.PageDefinition{
		{
			PageType:    "research",
			Title:       "Research",
			Slug:        "research",
			ShowInNav:   true,
			NavOrder:    20,
			Description: "Displays Google Scholar publications",
		},
	}
}

func (p *ScholarPlugin) ensureScholar(settings map[string]string) {
	if p.sch == nil {
		profileCache := settings["profile_cache"]
		articleCache := settings["article_cache"]
		if profileCache == "" {
			profileCache = "profiles.json"
		}
		if articleCache == "" {
			articleCache = "articles.json"
		}
		p.sch = scholarlib.New(profileCache, articleCache)
	}
}

func (p *ScholarPlugin) RenderPage(ctx *gplugin.HookContext, pageType string) (string, gin.H) {
	if pageType != "research" {
		return "", nil
	}

	settings := ctx.Settings
	scholarID := settings["scholar_id"]
	if scholarID == "" {
		return "page_research.html", gin.H{
			"errors": "Google Scholar ID not configured. Set it in the Scholar plugin settings.",
		}
	}

	limitStr := settings["article_limit"]
	limit := 50
	if limitStr != "" {
		fmt.Sscanf(limitStr, "%d", &limit)
	}

	p.ensureScholar(settings)

	articles, err := p.sch.QueryProfileWithMemoryCache(scholarID, limit)
	data := gin.H{}
	if err == nil {
		sortArticlesByDateDesc(articles)
		p.sch.SaveCache(settings["profile_cache"], settings["article_cache"])
		data["articles"] = articles
	} else {
		log.Printf("Scholar query failed: %v", err)
		data["articles"] = make([]*scholarlib.Article, 0)
		data["errors"] = err.Error()
	}

	return "page_research.html", data
}

func (p *ScholarPlugin) ScheduledJobs() []gplugin.ScheduledJob {
	return []gplugin.ScheduledJob{
		{
			Name:     "scholar-cache-refresh",
			Interval: 24 * time.Hour,
			Run: func(db *gorm.DB, settings map[string]string) error {
				scholarID := settings["scholar_id"]
				if scholarID == "" || settings["enabled"] != "true" {
					return nil
				}
				p.ensureScholar(settings)
				limit := 50
				fmt.Sscanf(settings["article_limit"], "%d", &limit)
				_, err := p.sch.QueryProfileWithMemoryCache(scholarID, limit)
				if err == nil {
					p.sch.SaveCache(settings["profile_cache"], settings["article_cache"])
				}
				return err
			},
		},
	}
}

func sortArticlesByDateDesc(articles []*scholarlib.Article) {
	sort.Slice(articles, func(i, j int) bool {
		if articles[i].Year != articles[j].Year {
			return articles[i].Year > articles[j].Year
		}
		if articles[i].Month != articles[j].Month {
			return articles[i].Month > articles[j].Month
		}
		return articles[i].Day > articles[j].Day
	})
}
