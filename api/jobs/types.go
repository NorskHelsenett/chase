package jobs

import (
	"time"

	"gorm.io/gorm"
)

// JobLog represents a persistent job execution record
type JobLog struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	JobName   string    `gorm:"type:varchar(100);index" json:"job_name"`
	Status    string    `gorm:"type:varchar(20);index" json:"status"` // running, completed, failed, cancelled
	Trigger   string    `gorm:"type:varchar(50)" json:"trigger"`     // manual, schedule, event
	Total     int       `json:"total"`
	Completed int       `json:"completed"`
	Failed    int       `json:"failed"`
	Logs      string    `gorm:"type:text" json:"logs"`
	Error     string    `gorm:"type:text" json:"error,omitempty"`
	StartTime time.Time `gorm:"index" json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// JobDefinition describes a registered job type
type JobDefinition struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Schedule    string `json:"schedule,omitempty"` // cron-like: "every 6h", "every 24h", etc.
	Running     bool   `json:"running"`
}

// AutoMigrate creates the job tables
func AutoMigrate(db *gorm.DB) error {
	return db.AutoMigrate(&JobLog{})
}
