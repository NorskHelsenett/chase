package types

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model     `json:"-"`
	APIToken       string      `json:"-"`
	Name           string      `json:"name"`
	Email          string      `json:"email"`
	VisitedServers IntegerList `json:"visited_servers" gorm:"type:json"`
}

// IntegerList is a custom type for storing an array of integers in the database as JSON
type IntegerList []int

// Scan implements the sql.Scanner interface for IntegerList
func (il *IntegerList) Scan(value interface{}) error {
	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("failed to unmarshal IntegerList value")
	}

	if len(bytes) == 0 {
		*il = make([]int, 0)
		return nil
	}

	return json.Unmarshal(bytes, il)
}

// Value implements the driver.Valuer interface for IntegerList
func (il IntegerList) Value() (driver.Value, error) {
	if il == nil {
		return nil, nil
	}
	return json.Marshal(il)
}
