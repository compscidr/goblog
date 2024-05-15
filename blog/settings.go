package blog

type Setting struct {
	ID    uint   `gorm:"primary_key"`
	Key   string `json:"key"`
	Value string `json:"value"`
}
