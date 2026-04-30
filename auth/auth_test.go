package auth_test

import (
	"net/http/httptest"
	"testing"

	"goblog/auth"

	"github.com/gin-contrib/sessions"
	"github.com/gin-contrib/sessions/cookie"
	"github.com/gin-gonic/gin"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func newCtx() *gin.Context {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	r := gin.Default()
	store := cookie.NewStore([]byte("test"))
	r.Use(sessions.Sessions("session", store))
	c.Request = httptest.NewRequest("GET", "/", nil)
	r.HandleContext(c)
	return c
}

func newAuth(t *testing.T) (*auth.Auth, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.AutoMigrate(&auth.BlogUser{}, &auth.AdminUser{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	a := auth.New(db, "test")
	return &a, db
}

func TestIsAdmin_NoAdminUser_ReturnsFalse(t *testing.T) {
	a, _ := newAuth(t)
	if a.IsAdmin(newCtx()) {
		t.Fatal("IsAdmin must be false on a fresh install with no admin_users row; otherwise anonymous traffic is treated as admin")
	}
}

func TestIsWizardMode_NoAdminUser_ReturnsTrue(t *testing.T) {
	a, _ := newAuth(t)
	if !a.IsWizardMode(newCtx()) {
		t.Fatal("IsWizardMode must be true when no admin_users row exists")
	}
}

func TestIsWizardMode_WithAdminUser_ReturnsFalse(t *testing.T) {
	a, db := newAuth(t)
	user := auth.BlogUser{ID: 1, Login: "admin"}
	db.Create(&user)
	db.Create(&auth.AdminUser{BlogUserID: user.ID, BlogUser: user})
	if a.IsWizardMode(newCtx()) {
		t.Fatal("IsWizardMode must be false once an admin_users row exists")
	}
}
