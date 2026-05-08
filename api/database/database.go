package database

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/norskhelsenett/chase/types"
	"github.com/norskhelsenett/chase/utils"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB

func InitDatabase() error {
	// Configure GORM logger
	logLevel := logger.Error
	if utils.GetEnv("GORM_LOG_LEVEL", "error") == "info" {
		logLevel = logger.Info
	}

	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             1000 * time.Millisecond,
			LogLevel:                  logLevel,
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	dsn, err := buildDSN()
	if err != nil {
		return err
	}

	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: newLogger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	if sqlDB, err := db.DB(); err == nil {
		sqlDB.SetMaxOpenConns(20)
		sqlDB.SetMaxIdleConns(5)
		sqlDB.SetConnMaxLifetime(30 * time.Minute)
	}

	if err := db.AutoMigrate(&types.User{}); err != nil {
		return fmt.Errorf("failed to auto migrate: %v", err)
	}

	return nil
}

// buildDSN constructs a Postgres DSN. Prefers DATABASE_URL when set,
// otherwise composes from PG* env vars (matching the convention used by
// the CloudNativePG operator's generated app secret).
func buildDSN() (string, error) {
	if dsn := os.Getenv("DATABASE_URL"); dsn != "" {
		return dsn, nil
	}

	host := utils.GetEnv("PGHOST", "")
	if host == "" {
		return "", fmt.Errorf("DATABASE_URL or PGHOST must be set")
	}
	port := utils.GetEnv("PGPORT", "5432")
	user := utils.GetEnv("PGUSER", "chase")
	password := os.Getenv("PGPASSWORD")
	dbName := utils.GetEnv("PGDATABASE", "chase")
	sslMode := utils.GetEnv("PGSSLMODE", "require")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbName, sslMode), nil
}

func GetDB() *gorm.DB {
	return db
}
