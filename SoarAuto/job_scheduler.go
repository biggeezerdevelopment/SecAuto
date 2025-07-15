package main

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// ScheduleType represents the type of schedule
type ScheduleType string

const (
	ScheduleTypeCron      ScheduleType = "cron"
	ScheduleTypeInterval  ScheduleType = "interval"
	ScheduleTypeOnce      ScheduleType = "once"
	ScheduleTypeRecurring ScheduleType = "recurring"
)

// ScheduleStatus represents the status of a schedule
type ScheduleStatus string

const (
	ScheduleStatusActive  ScheduleStatus = "active"
	ScheduleStatusPaused  ScheduleStatus = "paused"
	ScheduleStatusDeleted ScheduleStatus = "deleted"
	ScheduleStatusExpired ScheduleStatus = "expired"
)

// JobSchedule represents a scheduled job
type JobSchedule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	ScheduleType    ScheduleType           `json:"schedule_type"`
	CronExpression  string                 `json:"cron_expression,omitempty"`
	IntervalSeconds int                    `json:"interval_seconds,omitempty"`
	StartTime       *time.Time             `json:"start_time,omitempty"`
	EndTime         *time.Time             `json:"end_time,omitempty"`
	NextRun         *time.Time             `json:"next_run,omitempty"`
	LastRun         *time.Time             `json:"last_run,omitempty"`
	RunCount        int                    `json:"run_count"`
	MaxRuns         int                    `json:"max_runs,omitempty"`
	Status          ScheduleStatus         `json:"status"`
	Playbook        []interface{}          `json:"playbook"`
	Context         map[string]interface{} `json:"context"`
	Priority        int                    `json:"priority"`
	Tags            []string               `json:"tags"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	CreatedBy       string                 `json:"created_by"`
}

// SchedulerConfig is defined in config.go

// JobScheduler manages scheduled jobs
type JobScheduler struct {
	config         *SchedulerConfig
	cronScheduler  *cron.Cron
	schedules      map[string]*JobSchedule
	mutex          sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	jobStore       JobStoreInterface
	clusterManager *ClusterManager
	logger         *StructuredLogger
	server         *SecAutoServer
}

// NewJobScheduler creates a new job scheduler
func NewJobScheduler(config *SchedulerConfig, server *SecAutoServer) (*JobScheduler, error) {
	ctx, cancel := context.WithCancel(context.Background())

	scheduler := &JobScheduler{
		config:         config,
		cronScheduler:  cron.New(cron.WithSeconds()),
		schedules:      make(map[string]*JobSchedule),
		ctx:            ctx,
		cancel:         cancel,
		jobStore:       server.jobManager.store,
		clusterManager: server.clusterManager,
		logger:         logger,
		server:         server,
	}

	// Start the cron scheduler
	scheduler.cronScheduler.Start()

	// Load existing schedules from database
	if err := scheduler.loadSchedules(); err != nil {
		return nil, fmt.Errorf("failed to load schedules: %v", err)
	}

	// Start background tasks
	go scheduler.startBackgroundTasks()

	scheduler.logger.Info("Job scheduler started", map[string]interface{}{
		"component": "job_scheduler",
		"enabled":   config.Enabled,
		"schedules": len(scheduler.schedules),
	})

	return scheduler, nil
}

// loadSchedules loads schedules from the database
func (js *JobScheduler) loadSchedules() error {
	schedules := js.jobStore.ListSchedules("", 0) // Load all schedules

	for _, schedule := range schedules {
		if schedule.Status == ScheduleStatusActive {
			if err := js.addScheduleToCron(schedule); err != nil {
				js.logger.Error("Failed to add schedule to cron", map[string]interface{}{
					"component":   "job_scheduler",
					"schedule_id": schedule.ID,
					"error":       err.Error(),
				})
				continue
			}
		}
		js.schedules[schedule.ID] = schedule
	}

	return nil
}

// addScheduleToCron adds a schedule to the cron scheduler
func (js *JobScheduler) addScheduleToCron(schedule *JobSchedule) error {
	switch schedule.ScheduleType {
	case ScheduleTypeCron:
		if schedule.CronExpression == "" {
			return fmt.Errorf("cron expression is required for cron schedule type")
		}

		entryID, err := js.cronScheduler.AddFunc(schedule.CronExpression, func() {
			js.executeScheduledJob(schedule)
		})
		if err != nil {
			return fmt.Errorf("failed to add cron schedule: %v", err)
		}

		// Store entry ID for later removal
		schedule.ID = fmt.Sprintf("%d", entryID)

	case ScheduleTypeInterval:
		if schedule.IntervalSeconds <= 0 {
			return fmt.Errorf("interval must be greater than 0")
		}

		entryID, err := js.cronScheduler.AddFunc(fmt.Sprintf("@every %ds", schedule.IntervalSeconds), func() {
			js.executeScheduledJob(schedule)
		})
		if err != nil {
			return fmt.Errorf("failed to add interval schedule: %v", err)
		}

		schedule.ID = fmt.Sprintf("%d", entryID)

	case ScheduleTypeOnce:
		if schedule.StartTime == nil {
			return fmt.Errorf("start time is required for once schedule type")
		}

		// Calculate delay until start time
		delay := time.Until(*schedule.StartTime)
		if delay < 0 {
			return fmt.Errorf("start time is in the past")
		}

		time.AfterFunc(delay, func() {
			js.executeScheduledJob(schedule)
		})

	case ScheduleTypeRecurring:
		if schedule.IntervalSeconds <= 0 {
			return fmt.Errorf("interval must be greater than 0 for recurring schedule")
		}

		entryID, err := js.cronScheduler.AddFunc(fmt.Sprintf("@every %ds", schedule.IntervalSeconds), func() {
			js.executeScheduledJob(schedule)
		})
		if err != nil {
			return fmt.Errorf("failed to add recurring schedule: %v", err)
		}

		schedule.ID = fmt.Sprintf("%d", entryID)
	}

	return nil
}

// executeScheduledJob executes a scheduled job
func (js *JobScheduler) executeScheduledJob(schedule *JobSchedule) {
	js.logger.Info("Executing scheduled job", map[string]interface{}{
		"component":     "job_scheduler",
		"schedule_id":   schedule.ID,
		"schedule_name": schedule.Name,
	})

	// Check if schedule is still active
	if schedule.Status != ScheduleStatusActive {
		return
	}

	// Check if we've exceeded max runs
	if schedule.MaxRuns > 0 && schedule.RunCount >= schedule.MaxRuns {
		js.logger.Info("Schedule exceeded max runs", map[string]interface{}{
			"component":   "job_scheduler",
			"schedule_id": schedule.ID,
			"run_count":   schedule.RunCount,
			"max_runs":    schedule.MaxRuns,
		})
		schedule.Status = ScheduleStatusExpired
		js.updateSchedule(schedule)
		return
	}

	// Check if we're past the end time
	if schedule.EndTime != nil && time.Now().After(*schedule.EndTime) {
		js.logger.Info("Schedule past end time", map[string]interface{}{
			"component":   "job_scheduler",
			"schedule_id": schedule.ID,
			"end_time":    schedule.EndTime,
		})
		schedule.Status = ScheduleStatusExpired
		js.updateSchedule(schedule)
		return
	}

	// Submit job to queue
	var jobID string
	var err error

	if js.clusterManager != nil {
		// Submit to distributed queue
		jobID, err = js.clusterManager.SubmitJob(schedule.Playbook, schedule.Context)
	} else {
		// Submit to local job manager
		jobID = js.server.jobManager.SubmitJob(schedule.Playbook, schedule.Context)
	}

	if err != nil {
		js.logger.Error("Failed to submit scheduled job", map[string]interface{}{
			"component":   "job_scheduler",
			"schedule_id": schedule.ID,
			"error":       err.Error(),
		})
		return
	}

	// Update schedule statistics
	now := time.Now()
	schedule.LastRun = &now
	schedule.RunCount++
	schedule.NextRun = js.calculateNextRun(schedule)

	js.updateSchedule(schedule)

	js.logger.Info("Scheduled job submitted successfully", map[string]interface{}{
		"component":   "job_scheduler",
		"schedule_id": schedule.ID,
		"job_id":      jobID,
		"run_count":   schedule.RunCount,
	})
}

// calculateNextRun calculates the next run time for a schedule
func (js *JobScheduler) calculateNextRun(schedule *JobSchedule) *time.Time {
	switch schedule.ScheduleType {
	case ScheduleTypeCron:
		if schedule.CronExpression == "" {
			return nil
		}

		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		sched, err := parser.Parse(schedule.CronExpression)
		if err != nil {
			return nil
		}

		next := sched.Next(time.Now())
		return &next

	case ScheduleTypeInterval, ScheduleTypeRecurring:
		if schedule.IntervalSeconds <= 0 {
			return nil
		}

		next := time.Now().Add(time.Duration(schedule.IntervalSeconds) * time.Second)
		return &next

	case ScheduleTypeOnce:
		// One-time schedules don't have a next run
		return nil
	}

	return nil
}

// CreateSchedule creates a new scheduled job
func (js *JobScheduler) CreateSchedule(schedule *JobSchedule) error {
	// Validate schedule
	if err := js.validateSchedule(schedule); err != nil {
		return fmt.Errorf("invalid schedule: %v", err)
	}

	// Set default values
	if schedule.ID == "" {
		schedule.ID = generateScheduleID()
	}
	if schedule.Status == "" {
		schedule.Status = ScheduleStatusActive
	}
	if schedule.CreatedAt.IsZero() {
		schedule.CreatedAt = time.Now()
	}
	schedule.UpdatedAt = time.Now()

	// Calculate next run time
	schedule.NextRun = js.calculateNextRun(schedule)

	// Add to cron scheduler if active
	if schedule.Status == ScheduleStatusActive {
		if err := js.addScheduleToCron(schedule); err != nil {
			return fmt.Errorf("failed to add schedule to cron: %v", err)
		}
	}

	// Store in memory and database
	js.mutex.Lock()
	js.schedules[schedule.ID] = schedule
	js.mutex.Unlock()

	if err := js.jobStore.SaveSchedule(schedule); err != nil {
		return fmt.Errorf("failed to save schedule: %v", err)
	}

	js.logger.Info("Schedule created", map[string]interface{}{
		"component":     "job_scheduler",
		"schedule_id":   schedule.ID,
		"schedule_name": schedule.Name,
		"schedule_type": schedule.ScheduleType,
	})

	return nil
}

// UpdateSchedule updates an existing schedule
func (js *JobScheduler) UpdateSchedule(schedule *JobSchedule) error {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	existing, exists := js.schedules[schedule.ID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", schedule.ID)
	}

	// Update fields
	existing.Name = schedule.Name
	existing.Description = schedule.Description
	existing.ScheduleType = schedule.ScheduleType
	existing.CronExpression = schedule.CronExpression
	existing.IntervalSeconds = schedule.IntervalSeconds
	existing.StartTime = schedule.StartTime
	existing.EndTime = schedule.EndTime
	existing.MaxRuns = schedule.MaxRuns
	existing.Status = schedule.Status
	existing.Playbook = schedule.Playbook
	existing.Context = schedule.Context
	existing.Priority = schedule.Priority
	existing.Tags = schedule.Tags
	existing.UpdatedAt = time.Now()

	// Recalculate next run
	existing.NextRun = js.calculateNextRun(existing)

	// Update in database
	if err := js.jobStore.UpdateSchedule(existing); err != nil {
		return fmt.Errorf("failed to update schedule: %v", err)
	}

	js.logger.Info("Schedule updated", map[string]interface{}{
		"component":     "job_scheduler",
		"schedule_id":   existing.ID,
		"schedule_name": existing.Name,
	})

	return nil
}

// DeleteSchedule deletes a schedule
func (js *JobScheduler) DeleteSchedule(scheduleID string) error {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	schedule, exists := js.schedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	// Mark as deleted
	schedule.Status = ScheduleStatusDeleted
	schedule.UpdatedAt = time.Now()

	// Update in database
	if err := js.jobStore.UpdateSchedule(schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %v", err)
	}

	// Remove from memory
	delete(js.schedules, scheduleID)

	js.logger.Info("Schedule deleted", map[string]interface{}{
		"component":   "job_scheduler",
		"schedule_id": scheduleID,
	})

	return nil
}

// PauseSchedule pauses a schedule
func (js *JobScheduler) PauseSchedule(scheduleID string) error {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	schedule, exists := js.schedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	schedule.Status = ScheduleStatusPaused
	schedule.UpdatedAt = time.Now()

	if err := js.jobStore.UpdateSchedule(schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %v", err)
	}

	js.logger.Info("Schedule paused", map[string]interface{}{
		"component":   "job_scheduler",
		"schedule_id": scheduleID,
	})

	return nil
}

// ResumeSchedule resumes a paused schedule
func (js *JobScheduler) ResumeSchedule(scheduleID string) error {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	schedule, exists := js.schedules[scheduleID]
	if !exists {
		return fmt.Errorf("schedule not found: %s", scheduleID)
	}

	schedule.Status = ScheduleStatusActive
	schedule.UpdatedAt = time.Now()

	// Recalculate next run
	schedule.NextRun = js.calculateNextRun(schedule)

	if err := js.jobStore.UpdateSchedule(schedule); err != nil {
		return fmt.Errorf("failed to update schedule: %v", err)
	}

	js.logger.Info("Schedule resumed", map[string]interface{}{
		"component":   "job_scheduler",
		"schedule_id": scheduleID,
	})

	return nil
}

// GetSchedule retrieves a schedule by ID
func (js *JobScheduler) GetSchedule(scheduleID string) (*JobSchedule, bool) {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	schedule, exists := js.schedules[scheduleID]
	return schedule, exists
}

// ListSchedules lists all schedules with optional filtering
func (js *JobScheduler) ListSchedules(status ScheduleStatus, limit int) []*JobSchedule {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	var schedules []*JobSchedule
	count := 0

	for _, schedule := range js.schedules {
		if status != "" && schedule.Status != status {
			continue
		}

		schedules = append(schedules, schedule)
		count++

		if limit > 0 && count >= limit {
			break
		}
	}

	return schedules
}

// GetSchedulerStats returns scheduler statistics
func (js *JobScheduler) GetSchedulerStats() map[string]interface{} {
	js.mutex.RLock()
	defer js.mutex.RUnlock()

	total := len(js.schedules)
	active := 0
	paused := 0
	expired := 0

	for _, schedule := range js.schedules {
		switch schedule.Status {
		case ScheduleStatusActive:
			active++
		case ScheduleStatusPaused:
			paused++
		case ScheduleStatusExpired:
			expired++
		}
	}

	return map[string]interface{}{
		"total_schedules":   total,
		"active_schedules":  active,
		"paused_schedules":  paused,
		"expired_schedules": expired,
		"next_runs":         js.getNextRuns(),
	}
}

// getNextRuns returns the next few scheduled runs
func (js *JobScheduler) getNextRuns() []map[string]interface{} {
	var nextRuns []map[string]interface{}

	for _, schedule := range js.schedules {
		if schedule.Status == ScheduleStatusActive && schedule.NextRun != nil {
			nextRuns = append(nextRuns, map[string]interface{}{
				"schedule_id":   schedule.ID,
				"schedule_name": schedule.Name,
				"next_run":      schedule.NextRun,
			})
		}
	}

	return nextRuns
}

// validateSchedule validates a schedule
func (js *JobScheduler) validateSchedule(schedule *JobSchedule) error {
	if schedule.Name == "" {
		return fmt.Errorf("schedule name is required")
	}

	if len(schedule.Playbook) == 0 {
		return fmt.Errorf("playbook is required")
	}

	switch schedule.ScheduleType {
	case ScheduleTypeCron:
		if schedule.CronExpression == "" {
			return fmt.Errorf("cron expression is required for cron schedule type")
		}

		// Validate cron expression
		parser := cron.NewParser(cron.Second | cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
		if _, err := parser.Parse(schedule.CronExpression); err != nil {
			return fmt.Errorf("invalid cron expression: %v", err)
		}

	case ScheduleTypeInterval, ScheduleTypeRecurring:
		if schedule.IntervalSeconds <= 0 {
			return fmt.Errorf("interval must be greater than 0")
		}

	case ScheduleTypeOnce:
		if schedule.StartTime == nil {
			return fmt.Errorf("start time is required for once schedule type")
		}

		if time.Now().After(*schedule.StartTime) {
			return fmt.Errorf("start time cannot be in the past")
		}

	default:
		return fmt.Errorf("invalid schedule type: %s", schedule.ScheduleType)
	}

	return nil
}

// updateSchedule updates a schedule in the database
func (js *JobScheduler) updateSchedule(schedule *JobSchedule) {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	schedule.UpdatedAt = time.Now()

	if err := js.jobStore.UpdateSchedule(schedule); err != nil {
		js.logger.Error("Failed to update schedule", map[string]interface{}{
			"component":   "job_scheduler",
			"schedule_id": schedule.ID,
			"error":       err.Error(),
		})
	}
}

// startBackgroundTasks starts background maintenance tasks
func (js *JobScheduler) startBackgroundTasks() {
	// Cleanup expired schedules
	ticker := time.NewTicker(time.Duration(js.config.CleanupInterval) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-js.ctx.Done():
			return
		case <-ticker.C:
			js.cleanupExpiredSchedules()
		}
	}
}

// cleanupExpiredSchedules removes expired schedules
func (js *JobScheduler) cleanupExpiredSchedules() {
	js.mutex.Lock()
	defer js.mutex.Unlock()

	now := time.Now()
	expiredCount := 0

	for id, schedule := range js.schedules {
		// Check if schedule has expired
		if schedule.EndTime != nil && now.After(*schedule.EndTime) {
			schedule.Status = ScheduleStatusExpired
			schedule.UpdatedAt = now

			if err := js.jobStore.UpdateSchedule(schedule); err != nil {
				js.logger.Error("Failed to update expired schedule", map[string]interface{}{
					"component":   "job_scheduler",
					"schedule_id": id,
					"error":       err.Error(),
				})
			}
			expiredCount++
		}

		// Check if schedule exceeded max runs
		if schedule.MaxRuns > 0 && schedule.RunCount >= schedule.MaxRuns {
			schedule.Status = ScheduleStatusExpired
			schedule.UpdatedAt = now

			if err := js.jobStore.UpdateSchedule(schedule); err != nil {
				js.logger.Error("Failed to update maxed-out schedule", map[string]interface{}{
					"component":   "job_scheduler",
					"schedule_id": id,
					"error":       err.Error(),
				})
			}
			expiredCount++
		}
	}

	if expiredCount > 0 {
		js.logger.Info("Cleaned up expired schedules", map[string]interface{}{
			"component":     "job_scheduler",
			"expired_count": expiredCount,
		})
	}
}

// Close shuts down the job scheduler
func (js *JobScheduler) Close() error {
	js.cancel()

	// Stop cron scheduler
	js.cronScheduler.Stop()

	js.logger.Info("Job scheduler stopped", map[string]interface{}{
		"component": "job_scheduler",
		"schedules": len(js.schedules),
	})

	return nil
}

// Helper function to generate schedule ID
func generateScheduleID() string {
	return fmt.Sprintf("schedule_%d", time.Now().UnixNano())
}
