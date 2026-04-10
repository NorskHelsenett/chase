package jobs

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/norskhelsenett/chase/database"
)

// JobFunc is the function signature that all jobs must implement.
// It receives a context and a log writer, and returns (total, completed, failed, error).
type JobFunc func(ctx context.Context, appendLog func(string)) (total, completed, failed int, err error)

type registeredJob struct {
	definition JobDefinition
	run        JobFunc
	interval   time.Duration
	cancelFunc context.CancelFunc
}

var (
	registry   = make(map[string]*registeredJob)
	registryMu sync.RWMutex
	runningMu  sync.Mutex
	runningSet = make(map[string]uint) // jobName -> jobLogID
)

// Register adds a job to the scheduler. Schedule format: "every 6h", "every 30m", etc.
func Register(name, description, schedule string, fn JobFunc) {
	registryMu.Lock()
	defer registryMu.Unlock()

	interval := parseSchedule(schedule)
	registry[name] = &registeredJob{
		definition: JobDefinition{
			Name:        name,
			Description: description,
			Schedule:    schedule,
		},
		run:      fn,
		interval: interval,
	}
}

// StartScheduler begins running all scheduled jobs in the background.
func StartScheduler() {
	registryMu.RLock()
	defer registryMu.RUnlock()

	for name, job := range registry {
		if job.interval > 0 {
			go scheduleLoop(name, job)
		}
	}
	log.Println("Job scheduler started")
}

func scheduleLoop(name string, job *registeredJob) {
	// Run immediately on startup
	executeJob(name, "schedule")

	ticker := time.NewTicker(job.interval)
	defer ticker.Stop()

	for range ticker.C {
		executeJob(name, "schedule")
	}
}

// RunManually triggers a job by name. Returns the job log ID or an error.
func RunManually(name string) (uint, error) {
	registryMu.RLock()
	_, exists := registry[name]
	registryMu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("job %q not found", name)
	}

	runningMu.Lock()
	if _, alreadyRunning := runningSet[name]; alreadyRunning {
		runningMu.Unlock()
		return 0, fmt.Errorf("job %q is already running", name)
	}
	runningMu.Unlock()

	return executeJob(name, "manual"), nil
}

// CancelJob cancels a running job by name.
func CancelJob(name string) error {
	registryMu.RLock()
	job, exists := registry[name]
	registryMu.RUnlock()

	if !exists {
		return fmt.Errorf("job %q not found", name)
	}

	runningMu.Lock()
	_, isRunning := runningSet[name]
	runningMu.Unlock()

	if !isRunning {
		return fmt.Errorf("job %q is not running", name)
	}

	if job.cancelFunc != nil {
		job.cancelFunc()
	}
	return nil
}

// ListJobs returns all registered job definitions with running status.
func ListJobs() []JobDefinition {
	registryMu.RLock()
	defer registryMu.RUnlock()

	runningMu.Lock()
	defer runningMu.Unlock()

	defs := make([]JobDefinition, 0, len(registry))
	for name, job := range registry {
		def := job.definition
		_, def.Running = runningSet[name]
		defs = append(defs, def)
	}
	return defs
}

// GetLogs returns job logs with optional filtering and pagination.
func GetLogs(jobName string, limit, offset int) ([]JobLog, int64, error) {
	db := database.GetDB()
	query := db.Model(&JobLog{})

	if jobName != "" {
		query = query.Where("job_name = ?", jobName)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var logs []JobLog
	err := query.Order("start_time DESC").Limit(limit).Offset(offset).Find(&logs).Error
	return logs, total, err
}

func executeJob(name, trigger string) uint {
	registryMu.RLock()
	job := registry[name]
	registryMu.RUnlock()

	runningMu.Lock()
	if _, already := runningSet[name]; already {
		runningMu.Unlock()
		return 0
	}

	db := database.GetDB()
	jobLog := JobLog{
		JobName:   name,
		Status:    "running",
		Trigger:   trigger,
		StartTime: time.Now(),
	}
	db.Create(&jobLog)

	runningSet[name] = jobLog.ID
	runningMu.Unlock()

	// Mark running on the definition
	registryMu.Lock()
	job.definition.Running = true
	registryMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	registryMu.Lock()
	job.cancelFunc = cancel
	registryMu.Unlock()

	var logBuf strings.Builder
	appendLog := func(line string) {
		ts := time.Now().Format("15:04:05")
		entry := fmt.Sprintf("[%s] %s\n", ts, line)
		logBuf.WriteString(entry)
		log.Printf("[job:%s] %s", name, line)
	}

	go func() {
		defer cancel()

		total, completed, failed, err := job.run(ctx, appendLog)

		jobLog.Total = total
		jobLog.Completed = completed
		jobLog.Failed = failed
		jobLog.Logs = logBuf.String()
		jobLog.EndTime = time.Now()

		if ctx.Err() == context.Canceled {
			jobLog.Status = "cancelled"
		} else if err != nil {
			jobLog.Status = "failed"
			jobLog.Error = err.Error()
			appendLog(fmt.Sprintf("Error: %v", err))
			jobLog.Logs = logBuf.String()
		} else {
			jobLog.Status = "completed"
		}

		db.Save(&jobLog)

		runningMu.Lock()
		delete(runningSet, name)
		runningMu.Unlock()

		registryMu.Lock()
		job.definition.Running = false
		job.cancelFunc = nil
		registryMu.Unlock()
	}()

	return jobLog.ID
}

func parseSchedule(schedule string) time.Duration {
	schedule = strings.TrimSpace(strings.ToLower(schedule))
	if !strings.HasPrefix(schedule, "every ") {
		return 0
	}
	durationStr := strings.TrimPrefix(schedule, "every ")
	d, err := time.ParseDuration(durationStr)
	if err != nil {
		log.Printf("Invalid schedule %q: %v", schedule, err)
		return 0
	}
	return d
}
