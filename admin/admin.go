package admin

import (
	"log"
	"net/http"
)

// Admin handles admin requests
type Admin struct {
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
