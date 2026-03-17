package plugin_test

import (
	"goblog/plugin"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type testPlugin struct {
	plugin.BasePlugin
}

func (p *testPlugin) Name() string        { return "test" }
func (p *testPlugin) DisplayName() string { return "Test Plugin" }
func (p *testPlugin) Version() string     { return "0.1.0" }

func (p *testPlugin) Settings() []plugin.SettingDefinition {
	return []plugin.SettingDefinition{
		{Key: "api_key", Type: "text", DefaultValue: "default123", Label: "API Key"},
	}
}

func (p *testPlugin) TemplateHead(ctx *plugin.HookContext) string {
	if key := ctx.Settings["api_key"]; key != "" {
		return "<!-- test-head:" + key + " -->"
	}
	return ""
}

func (p *testPlugin) TemplateFooter(ctx *plugin.HookContext) string {
	return "<!-- test-footer -->"
}

func (p *testPlugin) TemplateData(ctx *plugin.HookContext) gin.H {
	return gin.H{"greeting": "hello from test plugin"}
}

func TestRegistryBasics(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&plugin.PluginSetting{}); err != nil {
		t.Fatal(err)
	}

	reg := plugin.NewRegistry(db)
	tp := &testPlugin{}
	reg.Register(tp)

	if len(reg.Plugins()) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(reg.Plugins()))
	}
	if reg.Plugins()[0].Name() != "test" {
		t.Fatalf("expected plugin name 'test', got %q", reg.Plugins()[0].Name())
	}
}

func TestRegistryInit(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&plugin.PluginSetting{}); err != nil {
		t.Fatal(err)
	}

	reg := plugin.NewRegistry(db)
	reg.Register(&testPlugin{})
	if err := reg.Init(); err != nil {
		t.Fatalf("Init failed: %v", err)
	}

	// Check setting was seeded
	var setting plugin.PluginSetting
	db.Where("plugin_name = ? AND key = ?", "test", "api_key").First(&setting)
	if setting.Value != "default123" {
		t.Fatalf("expected default value 'default123', got %q", setting.Value)
	}
}

func TestRegistryInjectTemplateData(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&plugin.PluginSetting{}); err != nil {
		t.Fatal(err)
	}

	reg := plugin.NewRegistry(db)
	reg.Register(&testPlugin{})
	reg.Init()

	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request, _ = http.NewRequest("GET", "/", nil)

	data := gin.H{"title": "Test Page"}
	data = reg.InjectTemplateData(c, "home.html", data)

	// Check plugin data was injected
	plugins, ok := data["plugins"].(gin.H)
	if !ok {
		t.Fatal("expected plugins key in data")
	}
	testData, ok := plugins["test"].(gin.H)
	if !ok {
		t.Fatal("expected test plugin data")
	}
	if testData["greeting"] != "hello from test plugin" {
		t.Fatalf("expected greeting, got %v", testData["greeting"])
	}

	// Check head/footer HTML
	headHTML, ok := data["plugin_head_html"].(string)
	if !ok || headHTML == "" {
		t.Fatal("expected plugin_head_html")
	}
	if headHTML != "<!-- test-head:default123 -->" {
		t.Fatalf("unexpected head HTML: %q", headHTML)
	}

	footerHTML, ok := data["plugin_footer_html"].(string)
	if !ok || footerHTML != "<!-- test-footer -->" {
		t.Fatalf("unexpected footer HTML: %q", footerHTML)
	}
}

func TestGetAllSettings(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatal(err)
	}
	if err := db.AutoMigrate(&plugin.PluginSetting{}); err != nil {
		t.Fatal(err)
	}

	reg := plugin.NewRegistry(db)
	reg.Register(&testPlugin{})
	reg.Init()

	groups := reg.GetAllSettings()
	if len(groups) != 1 {
		t.Fatalf("expected 1 settings group, got %d", len(groups))
	}
	if groups[0].PluginName != "test" {
		t.Fatalf("expected plugin name 'test', got %q", groups[0].PluginName)
	}
	if groups[0].CurrentValues["api_key"] != "default123" {
		t.Fatalf("expected current value 'default123', got %q", groups[0].CurrentValues["api_key"])
	}
}
