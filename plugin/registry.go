package plugin

import (
	"goblog/blog"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// PluginSettingsGroup holds a plugin's setting definitions and current values
// for rendering in the admin settings page.
type PluginSettingsGroup struct {
	PluginName    string
	DisplayName   string
	Settings      []SettingDefinition
	CurrentValues map[string]string
}

// Registry manages all registered plugins.
type Registry struct {
	plugins []Plugin
	db      *gorm.DB
	mu      sync.RWMutex
	stopCh  chan struct{}
}

// NewRegistry creates a plugin registry.
func NewRegistry(db *gorm.DB) *Registry {
	return &Registry{
		db:     db,
		stopCh: make(chan struct{}),
	}
}

// UpdateDb updates the database reference (used after wizard setup).
func (r *Registry) UpdateDb(db *gorm.DB) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.db = db
}

// Register adds a compiled-in plugin to the registry.
func (r *Registry) Register(p Plugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins = append(r.plugins, p)
	log.Printf("Plugin registered: %s v%s", p.DisplayName(), p.Version())
}

// Plugins returns the list of registered plugins.
func (r *Registry) Plugins() []Plugin {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.plugins
}

// Init seeds plugin settings and calls OnInit for all plugins.
func (r *Registry) Init() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.db == nil {
		return nil
	}
	for _, p := range r.plugins {
		for _, s := range p.Settings() {
			fullKey := p.Name() + "." + s.Key
			setting := blog.Setting{
				Key:   fullKey,
				Type:  s.Type,
				Value: s.DefaultValue,
			}
			r.db.Where("key = ?", fullKey).FirstOrCreate(&setting)
		}
		if err := p.OnInit(r.db); err != nil {
			log.Printf("Plugin %s init error: %v", p.Name(), err)
			return err
		}
	}
	return nil
}

// StartScheduledJobs launches goroutines for all plugin scheduled jobs.
func (r *Registry) StartScheduledJobs() {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, p := range r.plugins {
		for _, job := range p.ScheduledJobs() {
			go func(p Plugin, job ScheduledJob) {
				ticker := time.NewTicker(job.Interval)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						settings := r.getPluginSettings(p.Name())
						if err := job.Run(r.db, settings); err != nil {
							log.Printf("Plugin %s job %s error: %v", p.Name(), job.Name, err)
						}
					case <-r.stopCh:
						return
					}
				}
			}(p, job)
		}
	}
}

// Stop gracefully shuts down scheduled jobs.
func (r *Registry) Stop() {
	close(r.stopCh)
}

// getPluginSettings returns a plugin's settings as a map with the
// namespace prefix stripped (e.g. "analytics.tracking_id" -> "tracking_id").
func (r *Registry) getPluginSettings(pluginName string) map[string]string {
	var settings []blog.Setting
	prefix := pluginName + "."
	r.db.Where("key LIKE ?", prefix+"%").Find(&settings)
	result := make(map[string]string)
	for _, s := range settings {
		result[s.Key[len(prefix):]] = s.Value
	}
	return result
}

// InjectTemplateData gathers data from all plugins and merges it into
// the template data map. Adds "plugins", "plugin_head_html", and
// "plugin_footer_html" keys.
func (r *Registry) InjectTemplateData(c *gin.Context, templateName string, data gin.H) gin.H {
	r.mu.RLock()
	defer r.mu.RUnlock()

	pluginsData := gin.H{}
	headHTML := ""
	footerHTML := ""

	for _, p := range r.plugins {
		settings := r.getPluginSettings(p.Name())
		ctx := &HookContext{
			GinContext: c,
			DB:         r.db,
			Settings:   settings,
			Template:   templateName,
			Data:       data,
		}

		if pData := p.TemplateData(ctx); pData != nil {
			pluginsData[p.Name()] = pData
		}
		headHTML += p.TemplateHead(ctx)
		footerHTML += p.TemplateFooter(ctx)
	}

	data["plugins"] = pluginsData
	data["plugin_head_html"] = headHTML
	data["plugin_footer_html"] = footerHTML
	return data
}

// GetAllSettings returns all plugin setting definitions grouped by plugin.
func (r *Registry) GetAllSettings() []PluginSettingsGroup {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var groups []PluginSettingsGroup
	for _, p := range r.plugins {
		if defs := p.Settings(); len(defs) > 0 {
			values := r.getPluginSettings(p.Name())
			groups = append(groups, PluginSettingsGroup{
				PluginName:    p.Name(),
				DisplayName:   p.DisplayName(),
				Settings:      defs,
				CurrentValues: values,
			})
		}
	}
	return groups
}

// Middleware returns a Gin middleware that stores the registry on the context.
func Middleware(registry *Registry) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("plugin_registry", registry)
		c.Next()
	}
}
