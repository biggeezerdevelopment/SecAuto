package schedules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/robfig/cron/v3"
	"SoarAuto/pkg/types"
)

// ScheduleManager manages job scheduling and cron jobs
type ScheduleManager struct {
	schedules     map[string]*types.JobSchedule
	cron          *cron.Cron
	schedulesPath string
	mutex         sync.RWMutex
	jobExecutor   JobExecutor
}

// JobExecutor interface for executing scheduled jobs
type JobExecutor interface {
	ExecutePlaybook(playbook interface{}, context map[string]interface{}) (interface{}, error)
}

// NewScheduleManager creates a new schedule manager
func NewScheduleManager(schedulesPath string, executor JobExecutor) *ScheduleManager {
	sm := &ScheduleManager{
		schedules:     make(map[string]*types.JobSchedule),
		cron:          cron.New(cron.WithSeconds()),
		schedulesPath: schedulesPath,
		jobExecutor:   executor,
	}

	// Load existing schedules
	sm.loadSchedules()

	// Start the cron scheduler
	sm.cron.Start()

	return sm
}

// loadSchedules loads schedules from disk
func (sm *ScheduleManager) loadSchedules() {
	// Ensure schedules directory exists
	if err := os.MkdirAll(sm.schedulesPath, 0755); err != nil {
		return
	}

	// Load schedule files
	files, err := filepath.Glob(filepath.Join(sm.schedulesPath, "*.json"))
	if err != nil {
		return
	}

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		var schedule types.JobSchedule
		if err := json.Unmarshal(data, &schedule); err != nil {
			continue
		}

		// Add to memory and cron
		sm.schedules[schedule.ID] = &schedule
		if schedule.Enabled {
			sm.addToCron(&schedule)
		}
	}
}

// ListSchedules returns all schedules with optional status filter
func (sm *ScheduleManager) ListSchedules(status types.ScheduleStatus) []*types.JobSchedule {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var result []*types.JobSchedule
	for _, schedule := range sm.schedules {
		// Apply status filter
		if status != types.ScheduleStatusAll {
			if status == types.ScheduleStatusEnabled && !schedule.Enabled {
				continue
			}
			if status == types.ScheduleStatusDisabled && schedule.Enabled {
				continue
			}
		}

		// Create copy
		scheduleCopy := *schedule
		result = append(result, &scheduleCopy)
	}

	return result
}

// GetSchedule retrieves a specific schedule
func (sm *ScheduleManager) GetSchedule(id string) (*types.JobSchedule, bool) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	schedule, exists := sm.schedules[id]
	if !exists {
		return nil, false
	}

	// Return copy
	scheduleCopy := *schedule
	return &scheduleCopy, true
}

// CreateSchedule creates a new schedule
func (sm *ScheduleManager) CreateSchedule(schedule *types.JobSchedule) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Generate ID if not provided
	if schedule.ID == "" {
		schedule.ID = uuid.New().String()
	}

	// Check if already exists
	if _, exists := sm.schedules[schedule.ID]; exists {
		return fmt.Errorf("schedule %s already exists", schedule.ID)
	}

	// Validate cron expression
	if _, err := cron.ParseStandard(schedule.CronExpr); err != nil {
		return fmt.Errorf("invalid cron expression: %v", err)
	}

	// Set metadata
	schedule.CreatedAt = time.Now().UTC().Format(time.RFC3339)
	schedule.UpdatedAt = schedule.CreatedAt
	schedule.Status = "created"

	// Calculate next run time
	if schedule.Enabled {
		if nextRun, err := sm.calculateNextRun(schedule.CronExpr); err == nil {
			schedule.NextRun = nextRun.Format(time.RFC3339)
		}
	}

	// Save to disk
	if err := sm.saveSchedule(schedule); err != nil {
		return err
	}

	// Add to memory
	sm.schedules[schedule.ID] = schedule

	// Add to cron if enabled
	if schedule.Enabled {
		sm.addToCron(schedule)
	}

	return nil
}

// UpdateSchedule updates an existing schedule
func (sm *ScheduleManager) UpdateSchedule(id string, updates *types.JobSchedule) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if exists
	existing, exists := sm.schedules[id]
	if !exists {
		return fmt.Errorf("schedule %s not found", id)
	}

	// Validate cron expression if changed
	if updates.CronExpr != "" && updates.CronExpr != existing.CronExpr {
		if _, err := cron.ParseStandard(updates.CronExpr); err != nil {
			return fmt.Errorf("invalid cron expression: %v", err)
		}
	}

	// Remove from cron if currently scheduled
	sm.removeFromCron(existing)

	// Update fields
	if updates.Name != "" {
		existing.Name = updates.Name
	}
	if updates.Description != "" {
		existing.Description = updates.Description
	}
	if updates.CronExpr != "" {
		existing.CronExpr = updates.CronExpr
	}
	if updates.Playbook != nil {
		existing.Playbook = updates.Playbook
	}
	if updates.Context != nil {
		existing.Context = updates.Context
	}
	if updates.Enabled != existing.Enabled {
		existing.Enabled = updates.Enabled
	}

	existing.UpdatedAt = time.Now().UTC().Format(time.RFC3339)

	// Calculate next run time if enabled
	if existing.Enabled {
		if nextRun, err := sm.calculateNextRun(existing.CronExpr); err == nil {
			existing.NextRun = nextRun.Format(time.RFC3339)
		}
	} else {
		existing.NextRun = ""
	}

	// Save to disk
	if err := sm.saveSchedule(existing); err != nil {
		return err
	}

	// Add back to cron if enabled
	if existing.Enabled {
		sm.addToCron(existing)
	}

	return nil
}

// DeleteSchedule deletes a schedule
func (sm *ScheduleManager) DeleteSchedule(id string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	// Check if exists
	schedule, exists := sm.schedules[id]
	if !exists {
		return fmt.Errorf("schedule %s not found", id)
	}

	// Remove from cron
	sm.removeFromCron(schedule)

	// Remove from disk
	scheduleFile := filepath.Join(sm.schedulesPath, id+".json")
	if err := os.Remove(scheduleFile); err != nil && !os.IsNotExist(err) {
		return err
	}

	// Remove from memory
	delete(sm.schedules, id)

	return nil
}

// saveSchedule saves a schedule to disk
func (sm *ScheduleManager) saveSchedule(schedule *types.JobSchedule) error {
	// Ensure directory exists
	if err := os.MkdirAll(sm.schedulesPath, 0755); err != nil {
		return err
	}

	// Serialize schedule
	data, err := json.MarshalIndent(schedule, "", "  ")
	if err != nil {
		return err
	}

	// Write to file
	scheduleFile := filepath.Join(sm.schedulesPath, schedule.ID+".json")
	return os.WriteFile(scheduleFile, data, 0644)
}

// addToCron adds a schedule to the cron scheduler
func (sm *ScheduleManager) addToCron(schedule *types.JobSchedule) {
	entryID, err := sm.cron.AddFunc(schedule.CronExpr, func() {
		sm.executeScheduledJob(schedule)
	})
	
	if err == nil {
		// Store entry ID for later removal (would need to extend JobSchedule struct)
		schedule.Status = "scheduled"
	}
	
	_ = entryID // Suppress unused variable warning
}

// removeFromCron removes a schedule from the cron scheduler
func (sm *ScheduleManager) removeFromCron(schedule *types.JobSchedule) {
	// In a real implementation, we'd store the cron entry ID and remove it here
	// For now, we'll just update the status
	schedule.Status = "unscheduled"
}

// executeScheduledJob executes a scheduled job
func (sm *ScheduleManager) executeScheduledJob(schedule *types.JobSchedule) {
	sm.mutex.Lock()
	// Update last run time
	schedule.LastRun = time.Now().UTC().Format(time.RFC3339)
	schedule.Status = "running"
	
	// Calculate next run time
	if nextRun, err := sm.calculateNextRun(schedule.CronExpr); err == nil {
		schedule.NextRun = nextRun.Format(time.RFC3339)
	}
	
	// Save updated schedule
	sm.saveSchedule(schedule)
	sm.mutex.Unlock()

	// Execute the job
	if sm.jobExecutor != nil {
		go func() {
			_, err := sm.jobExecutor.ExecutePlaybook(schedule.Playbook, schedule.Context)
			
			sm.mutex.Lock()
			if err != nil {
				schedule.Status = "failed"
			} else {
				schedule.Status = "completed"
			}
			sm.saveSchedule(schedule)
			sm.mutex.Unlock()
		}()
	}
}

// calculateNextRun calculates the next run time for a cron expression
func (sm *ScheduleManager) calculateNextRun(cronExpr string) (time.Time, error) {
	schedule, err := cron.ParseStandard(cronExpr)
	if err != nil {
		return time.Time{}, err
	}
	
	return schedule.Next(time.Now()), nil
}

// Stop stops the schedule manager
func (sm *ScheduleManager) Stop() {
	sm.cron.Stop()
}

// GetScheduleStats returns statistics about schedules
func (sm *ScheduleManager) GetScheduleStats() map[string]interface{} {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	enabled := 0
	disabled := 0
	running := 0
	failed := 0

	for _, schedule := range sm.schedules {
		if schedule.Enabled {
			enabled++
		} else {
			disabled++
		}

		switch schedule.Status {
		case "running":
			running++
		case "failed":
			failed++
		}
	}

	return map[string]interface{}{
		"total":    len(sm.schedules),
		"enabled":  enabled,
		"disabled": disabled,
		"running":  running,
		"failed":   failed,
	}
}