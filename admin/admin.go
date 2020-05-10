package admin

import (
	. "goblog/auth"
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

	//check to see if user is logged in (todo add expiry)
	//can't do this until we publish a version with the auth module in it
	var existingUser BlogUser
	err := a.db.Where("access_token = ?", auth).First(&existingUser).Error
	if err != nil {
		http.Error(w, "Not authorized", http.StatusForbidden)
		return
	}

	log.Println(auth)
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")
	w.WriteHeader(http.StatusOK)
}
