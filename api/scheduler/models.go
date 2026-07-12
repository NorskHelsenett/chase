package scheduler

import "time"

// JobRunRecord is persisted in Postgres for run history.
type JobRunRecord struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	JobName   string    `gorm:"index;type:varchar(100)" json:"job_name"`
	Trigger   string    `gorm:"type:varchar(20)" json:"trigger"` // "scheduled" or "manual"
	Status    string    `gorm:"type:varchar(20)" json:"status"`
	StartedAt time.Time `json:"started_at"`
	EndedAt   time.Time `json:"ended_at"`
	Duration  float64   `json:"duration_seconds"`
	Summary   string    `gorm:"type:text" json:"summary"`
	Error     string    `gorm:"type:text" json:"error,omitempty"`
}
