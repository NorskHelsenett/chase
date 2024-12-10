package types

import "gorm.io/gorm"

type User struct {
	gorm.Model `json:"-"`
	APIToken   string `json:"-"`
	Name       string `json:"name"`
	Email      string `json:"email"`
}
