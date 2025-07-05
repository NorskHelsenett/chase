package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/norskhelsenett/chase/types"
	"github.com/norskhelsenett/chase/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func InitDatabase() error {
	dataFolder := utils.GetEnv("DATA_FOLDER", "/data")

	// Configure GORM logger
	logLevel := logger.Error // Only show error logs
	if utils.GetEnv("GORM_LOG_LEVEL", "error") == "info" {
		logLevel = logger.Info
	}

	// Create custom logger config
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags), // io writer
		logger.Config{
			SlowThreshold:             1000 * time.Millisecond, // Slow SQL threshold set to 1s (increased from 200ms)
			LogLevel:                  logLevel,                // Log level (Error by default to minimize logs)
			IgnoreRecordNotFoundError: true,                    // Ignore ErrRecordNotFound error
			Colorful:                  false,                   // Disable color
		},
	)

	var err error
	db, err = gorm.Open(sqlite.Open(filepath.Join(dataFolder, "chase.db?_loc=Local")), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	optimizeSQLiteForWrites(db)

	// Auto Migrate the schema
	if err := db.AutoMigrate(&types.User{}); err != nil {
		return fmt.Errorf("failed to auto migrate: %v", err)
	}

	return nil
}

func GetDB() *gorm.DB {
	return db
}

func optimizeSQLiteForWrites(db *gorm.DB) {
	db.Exec("PRAGMA journal_mode=WAL;")
	db.Exec("PRAGMA synchronous=NORMAL;")
	db.Exec("PRAGMA page_size=4096;")
	db.Exec("PRAGMA cache_size=-131072;")
	db.Exec("PRAGMA mmap_size=268435456;")
	db.Exec("PRAGMA temp_store=MEMORY;")
	db.Exec("PRAGMA journal_size_limit=33554432;")
	db.Exec("PRAGMA busy_timeout=5000;")
}
