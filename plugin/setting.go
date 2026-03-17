package plugin

// PluginSetting stores a plugin's configuration in its own table,
// separate from the blog's core Setting table.
type PluginSetting struct {
	PluginName string `gorm:"primaryKey" json:"plugin_name"`
	Key        string `gorm:"primaryKey" json:"key"`
	Value      string `json:"value"`
}
