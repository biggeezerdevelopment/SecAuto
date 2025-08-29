package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"SoarAuto/pkg/config"
	"SoarAuto/pkg/database"
	"SoarAuto/pkg/logger"

	_ "github.com/lib/pq"
)

// LegacyClient represents the old file-based client structure
type LegacyClient struct {
	ID               string                 `json:"id"`
	Name             string                 `json:"name"`
	Description      string                 `json:"description"`
	Active           bool                   `json:"active"`
	APIKeys          []string               `json:"api_keys"`
	EncryptionKeyID  string                 `json:"encryption_key_id"`
	RateLimits       *LegacyRateLimitConfig `json:"rate_limits,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        string                 `json:"created_at"`
	UpdatedAt        string                 `json:"updated_at"`
	LastAccessedAt   string                 `json:"last_accessed_at,omitempty"`
	IntegrationCount int                    `json:"integration_count"`
}

type LegacyRateLimitConfig struct {
	RequestsPerMinute int  `json:"requests_per_minute"`
	BurstLimit        int  `json:"burst_limit"`
	Enabled           bool `json:"enabled"`
}

func main() {
	fmt.Println("SecAuto Client Migration Tool")
	fmt.Println("=============================")

	// Load configuration
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create logger
	lgr := logger.NewStructuredLoggerWithConfig(
		logger.LogLevel(cfg.Logging.Level),
		"console",
		"",
		nil,
		nil,
		nil,
	)

	// Connect to PostgreSQL
	db, err := connectToDatabase(cfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create client repository
	clientRepo := database.NewClientRepository(db, lgr)

	// Check if clients table exists
	if !tableExists(db, "clients") {
		log.Fatal("Clients table does not exist. Please run migration 002 first.")
	}

	// Load legacy clients from file
	legacyClients, err := loadLegacyClients(cfg)
	if err != nil {
		log.Printf("Warning: Failed to load legacy clients: %v", err)
		fmt.Println("No legacy clients found or failed to load. Migration complete.")
		return
	}

	if len(legacyClients) == 0 {
		fmt.Println("No legacy clients found. Migration complete.")
		return
	}

	fmt.Printf("Found %d legacy clients to migrate\n", len(legacyClients))

	// Check if any clients already exist in database
	existingClients, err := clientRepo.ListClients(false, 0, 0)
	if err != nil {
		log.Fatalf("Failed to check existing clients: %v", err)
	}

	if len(existingClients) > 0 {
		fmt.Printf("Warning: Found %d existing clients in database.\n", len(existingClients))
		fmt.Print("Do you want to continue? This will skip existing clients. (y/N): ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" {
			fmt.Println("Migration cancelled.")
			return
		}
	}

	// Migrate each client
	migratedCount := 0
	skippedCount := 0

	for _, legacyClient := range legacyClients {
		// Check if client already exists
		_, err := clientRepo.GetClient(legacyClient.ID)
		if err == nil {
			fmt.Printf("Skipping existing client: %s (%s)\n", legacyClient.Name, legacyClient.ID)
			skippedCount++
			continue
		}

		// Convert legacy client to new format
		dbClient, err := convertLegacyClient(legacyClient)
		if err != nil {
			log.Printf("Failed to convert client %s: %v", legacyClient.ID, err)
			continue
		}

		// Create client in database
		err = clientRepo.CreateClient(dbClient)
		if err != nil {
			log.Printf("Failed to create client %s: %v", legacyClient.ID, err)
			continue
		}

		fmt.Printf("Migrated client: %s (%s)\n", dbClient.Name, dbClient.ID)
		migratedCount++
	}

	fmt.Printf("\nMigration Summary:\n")
	fmt.Printf("- Total legacy clients: %d\n", len(legacyClients))
	fmt.Printf("- Successfully migrated: %d\n", migratedCount)
	fmt.Printf("- Skipped (already exist): %d\n", skippedCount)

	if migratedCount > 0 {
		fmt.Printf("\nMigration completed successfully!\n")
		fmt.Printf("You can now update your client manager to use the database implementation.\n")

		// Ask about backing up legacy file
		fmt.Print("Do you want to backup the legacy clients file? (Y/n): ")
		var response string
		fmt.Scanln(&response)
		if response != "n" && response != "N" {
			err := backupLegacyFile(cfg)
			if err != nil {
				log.Printf("Failed to backup legacy file: %v", err)
			} else {
				fmt.Println("Legacy clients file backed up successfully.")
			}
		}
	}
}

func connectToDatabase(cfg *config.Config) (*sql.DB, error) {
	connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s sslmode=%s",
		cfg.Database.Postgres.Host,
		cfg.Database.Postgres.Port,
		cfg.Database.Postgres.Username,
		cfg.Database.Postgres.Database,
		cfg.Database.Postgres.SSLMode)

	if cfg.Database.Postgres.Password != "" {
		connStr += fmt.Sprintf(" password=%s", cfg.Database.Postgres.Password)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}

	if err = db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return db, nil
}

func tableExists(db *sql.DB, tableName string) bool {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.tables 
			WHERE table_schema = 'public' AND table_name = $1
		)`

	var exists bool
	err := db.QueryRow(query, tableName).Scan(&exists)
	return err == nil && exists
}

func loadLegacyClients(cfg *config.Config) (map[string]*LegacyClient, error) {
	clientsPath := filepath.Join("data", "clients", "clients_metadata.json")

	data, err := os.ReadFile(clientsPath)
	if err != nil {
		return nil, err
	}

	var clients map[string]*LegacyClient
	if err := json.Unmarshal(data, &clients); err != nil {
		return nil, fmt.Errorf("failed to unmarshal clients metadata: %v", err)
	}

	return clients, nil
}

func convertLegacyClient(legacy *LegacyClient) (*database.Client, error) {
	// Parse timestamps
	createdAt, err := time.Parse(time.RFC3339, legacy.CreatedAt)
	if err != nil {
		// Use current time if parsing fails
		createdAt = time.Now()
	}

	updatedAt, err := time.Parse(time.RFC3339, legacy.UpdatedAt)
	if err != nil {
		updatedAt = createdAt
	}

	var lastAccessedAt *time.Time
	if legacy.LastAccessedAt != "" {
		if t, err := time.Parse(time.RFC3339, legacy.LastAccessedAt); err == nil {
			lastAccessedAt = &t
		}
	}

	// Convert rate limits
	rateLimitRequestsPerMin := 100
	rateLimitBurstLimit := 20
	rateLimitEnabled := true

	if legacy.RateLimits != nil {
		rateLimitRequestsPerMin = legacy.RateLimits.RequestsPerMinute
		rateLimitBurstLimit = legacy.RateLimits.BurstLimit
		rateLimitEnabled = legacy.RateLimits.Enabled
	}

	// Ensure metadata exists
	metadata := legacy.Metadata
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &database.Client{
		ID:                      legacy.ID,
		Name:                    legacy.Name,
		Description:             legacy.Description,
		Active:                  legacy.Active,
		APIKeys:                 legacy.APIKeys,
		EncryptionKeyID:         legacy.EncryptionKeyID,
		RateLimitRequestsPerMin: rateLimitRequestsPerMin,
		RateLimitBurstLimit:     rateLimitBurstLimit,
		RateLimitEnabled:        rateLimitEnabled,
		Metadata:                metadata,
		CreatedAt:               createdAt,
		UpdatedAt:               updatedAt,
		LastAccessedAt:          lastAccessedAt,
		IntegrationCount:        legacy.IntegrationCount,
	}, nil
}

func backupLegacyFile(cfg *config.Config) error {
	sourcePath := filepath.Join("data", "clients", "clients_metadata.json")
	backupPath := fmt.Sprintf("%s.backup_%s", sourcePath, time.Now().Format("20060102_150405"))

	sourceData, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	return os.WriteFile(backupPath, sourceData, 0644)
}
