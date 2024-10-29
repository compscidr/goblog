package blog

type Setting struct {
	Key  string `gorm:"primary_key" json:"key"`
	Type string `json:"type"` // indicates whether it's a file, string, number, etc - mostly for the front-end of the
	// admin panel to know how to render the input field
	Value string `json:"value"`
}
