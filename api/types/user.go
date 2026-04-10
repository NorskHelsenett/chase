package types

import "gorm.io/gorm"

type User struct {
	gorm.Model `json:"-"`
	APIToken   string `json:"-"`
	Name       string `json:"name"`
	Email      string `json:"email"`
}

// IsAdmin returns true if the given email belongs to the first registered user.
func IsAdmin(db *gorm.DB, email string) bool {
	var first User
	if err := db.Order("id ASC").First(&first).Error; err != nil {
		return false
	}
	return first.Email == email
}
