package blog

type Setting struct {
	Key   string `gorm:"primary_key" json:"key"`
	Value string `json:"value"`
}
