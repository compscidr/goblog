package plugin

import "reflect"

// Symbols exports the plugin package types for the Yaegi interpreter,
// allowing dynamic plugins to use plugin.BasePlugin, plugin.HookContext, etc.
var Symbols = map[string]map[string]reflect.Value{
	"goblog/plugin/plugin": {
		"BasePlugin":       reflect.ValueOf((*BasePlugin)(nil)),
		"HookContext":      reflect.ValueOf((*HookContext)(nil)),
		"SettingDefinition": reflect.ValueOf((*SettingDefinition)(nil)),
		"ScheduledJob":     reflect.ValueOf((*ScheduledJob)(nil)),
	},
}
