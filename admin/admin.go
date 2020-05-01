package admin

import (
	"log"
	"net/http"

	"github.com/jinzhu/gorm"
)

// Admin handles admin requests
type Admin struct {
	db *gorm.DB
}

//New constructs an Admin API
func New(db *gorm.DB) Admin {
	api := Admin{db}
	return api
}

// AdminHandler handles admin requests
func (a Admin) AdminHandler(w http.ResponseWriter, r *http.Request) {
	auth, ok := r.Header["Authorization"]

	if !ok {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	log.Println(auth)
}
