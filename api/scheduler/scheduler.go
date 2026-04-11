package scheduler

import (
	"context"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"gorm.io/gorm"
)

// JobFunc is the signature for a scheduled job.
// It receives a context (for cancellation) and a progress reporter.
// Returns a summary string and an error.
type JobFunc func(ctx context.Context, progress func(msg string)) (summary string, err error)

// Schedule defines when a job runs.
type Schedule struct {
	Interval  time.Duration // Run every N duration
	TimeOfDay *TimeOfDay   // If set, run once daily at this time (overrides Interval)
}

// TimeOfDay represents a specific time of day.
type TimeOfDay struct {
	Hour   int
	Minute int
}

// JobStatus represents the current state of a job.
type JobStatus string

const (
	StatusIdle    JobStatus = "idle"
	StatusRunning JobStatus = "running"
	StatusSuccess JobStatus = "success"
	StatusFailed  JobStatus = "failed"
)

// Job is the in-memory representation of a registered job.
type Job struct {
	Name        string
	Description string
	Schedule    Schedule
	Fn          JobFunc

	// Runtime state (protected by Scheduler.mu)
	status   JobStatus
	lastRun  time.Time
	lastDur  time.Duration
	lastErr  string
	nextRun  time.Time
	progress string
	cancel   context.CancelFunc
}

// JobInfo is the JSON-serializable view of a job.
type JobInfo struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Status      JobStatus `json:"status"`
	LastRun     time.Time `json:"last_run,omitempty"`
	LastDur     float64   `json:"last_duration_seconds,omitempty"`
	LastErr     string    `json:"last_error,omitempty"`
	NextRun     time.Time `json:"next_run"`
	Progress    string    `json:"progress,omitempty"`
	Schedule    string    `json:"schedule"`
}

// Scheduler manages registered jobs, their schedules, and execution history.
type Scheduler struct {
	mu   sync.RWMutex
	jobs map[string]*Job
	db   *gorm.DB
	stop chan struct{}
}

// New creates a new Scheduler.
func New(db *gorm.DB) *Scheduler {
	return &Scheduler{
		jobs: make(map[string]*Job),
		db:   db,
		stop: make(chan struct{}),
	}
}

// Register adds a job to the scheduler.
func (s *Scheduler) Register(name, description string, sched Schedule, fn JobFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	j := &Job{
		Name:        name,
		Description: description,
		Schedule:    sched,
		Fn:          fn,
		status:      StatusIdle,
		nextRun:     calculateNextRun(sched, now),
	}
	s.jobs[name] = j
	log.Printf("Scheduler: registered job %q (next run: %s)", name, j.nextRun.Format("15:04:05"))
}

// Start begins the scheduler tick loop.
func (s *Scheduler) Start() {
	s.db.AutoMigrate(&JobRunRecord{})

	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.tick()
			case <-s.stop:
				return
			}
		}
	}()

	log.Println("Scheduler started")
}

// Stop halts the scheduler tick loop.
func (s *Scheduler) Stop() {
	close(s.stop)
}

func (s *Scheduler) tick() {
	now := time.Now()
	s.mu.RLock()
	var due []*Job
	for _, j := range s.jobs {
		if j.status != StatusRunning && now.After(j.nextRun) {
			due = append(due, j)
		}
	}
	s.mu.RUnlock()

	for _, j := range due {
		go s.runJob(j, false)
	}
}

func (s *Scheduler) runJob(j *Job, manual bool) {
	ctx, cancel := context.WithCancel(context.Background())

	s.mu.Lock()
	j.status = StatusRunning
	j.cancel = cancel
	j.progress = ""
	startedAt := time.Now()
	j.lastRun = startedAt
	s.mu.Unlock()

	trigger := "scheduled"
	if manual {
		trigger = "manual"
	}
	log.Printf("Scheduler: starting job %q (%s)", j.Name, trigger)

	progressFn := func(msg string) {
		s.mu.Lock()
		j.progress = msg
		s.mu.Unlock()
	}

	summary, err := j.Fn(ctx, progressFn)

	endedAt := time.Now()
	dur := endedAt.Sub(startedAt)

	s.mu.Lock()
	j.lastDur = dur
	if err != nil {
		j.status = StatusFailed
		j.lastErr = err.Error()
	} else {
		j.status = StatusSuccess
		j.lastErr = ""
	}
	j.nextRun = calculateNextRun(j.Schedule, endedAt)
	j.cancel = nil
	j.progress = ""
	s.mu.Unlock()

	cancel()

	log.Printf("Scheduler: job %q finished in %v (status: %s)", j.Name, dur, j.status)

	record := JobRunRecord{
		JobName:   j.Name,
		Trigger:   trigger,
		Status:    string(j.status),
		StartedAt: startedAt,
		EndedAt:   endedAt,
		Duration:  dur.Seconds(),
		Summary:   summary,
	}
	if err != nil {
		record.Error = err.Error()
	}
	s.db.Create(&record)

	s.pruneHistory(j.Name, 50)
}

// Trigger manually starts a job by name.
func (s *Scheduler) Trigger(name string) error {
	s.mu.RLock()
	j, ok := s.jobs[name]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("job %q not found", name)
	}

	s.mu.RLock()
	running := j.status == StatusRunning
	s.mu.RUnlock()
	if running {
		return fmt.Errorf("job %q is already running", name)
	}

	go s.runJob(j, true)
	return nil
}

// Cancel stops a running job.
func (s *Scheduler) Cancel(name string) error {
	s.mu.RLock()
	j, ok := s.jobs[name]
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("job %q not found", name)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if j.status != StatusRunning || j.cancel == nil {
		return fmt.Errorf("job %q is not running", name)
	}
	j.cancel()
	j.status = StatusFailed
	j.lastErr = "cancelled"
	j.progress = ""
	return nil
}

// ListJobs returns info about all registered jobs, sorted by name.
func (s *Scheduler) ListJobs() []JobInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	infos := make([]JobInfo, 0, len(s.jobs))
	for _, j := range s.jobs {
		infos = append(infos, JobInfo{
			Name:        j.Name,
			Description: j.Description,
			Status:      j.status,
			LastRun:     j.lastRun,
			LastDur:     j.lastDur.Seconds(),
			LastErr:     j.lastErr,
			NextRun:     j.nextRun,
			Progress:    j.progress,
			Schedule:    formatSchedule(j.Schedule),
		})
	}
	sort.Slice(infos, func(i, k int) bool { return infos[i].Name < infos[k].Name })
	return infos
}

// GetJob returns info about a single job.
func (s *Scheduler) GetJob(name string) (JobInfo, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	j, ok := s.jobs[name]
	if !ok {
		return JobInfo{}, false
	}
	return JobInfo{
		Name:        j.Name,
		Description: j.Description,
		Status:      j.status,
		LastRun:     j.lastRun,
		LastDur:     j.lastDur.Seconds(),
		LastErr:     j.lastErr,
		NextRun:     j.nextRun,
		Progress:    j.progress,
		Schedule:    formatSchedule(j.Schedule),
	}, true
}

func (s *Scheduler) pruneHistory(jobName string, keep int) {
	var count int64
	s.db.Model(&JobRunRecord{}).Where("job_name = ?", jobName).Count(&count)
	if count <= int64(keep) {
		return
	}
	s.db.Exec(`DELETE FROM job_run_records WHERE job_name = ? AND id NOT IN (
		SELECT id FROM job_run_records WHERE job_name = ? ORDER BY started_at DESC LIMIT ?
	)`, jobName, jobName, keep)
}

func calculateNextRun(sched Schedule, after time.Time) time.Time {
	if sched.TimeOfDay != nil {
		next := time.Date(after.Year(), after.Month(), after.Day(),
			sched.TimeOfDay.Hour, sched.TimeOfDay.Minute, 0, 0, after.Location())
		if !next.After(after) {
			next = next.Add(24 * time.Hour)
		}
		return next
	}
	return after.Add(sched.Interval)
}

func formatSchedule(sched Schedule) string {
	if sched.TimeOfDay != nil {
		return fmt.Sprintf("daily at %02d:%02d", sched.TimeOfDay.Hour, sched.TimeOfDay.Minute)
	}
	d := sched.Interval
	if d >= time.Hour {
		return fmt.Sprintf("every %dh", int(d.Hours()))
	}
	return fmt.Sprintf("every %dm", int(d.Minutes()))
}
