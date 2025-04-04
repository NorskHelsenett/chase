package database

import (
	"fmt"
	"path/filepath"

	"github.com/norskhelsenett/chase/types"
	"github.com/norskhelsenett/chase/utils"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDatabase() error {
	dataFolder := utils.GetEnv("DATA_FOLDER", "/data")

	var err error
	db, err = gorm.Open(sqlite.Open(filepath.Join(dataFolder, "fit.db?_loc=Local")), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("failed to connect database: %v", err)
	}

	optimizeSQLiteForNFSWrites(db)

	// Auto Migrate the schema
	if err := db.AutoMigrate(&types.User{}); err != nil {
		return fmt.Errorf("failed to auto migrate: %v", err)
	}

	return nil
}

func GetDB() *gorm.DB {
	return db
}

func optimizeSQLiteForNFSWrites(db *gorm.DB) {
	// Journal mode - WAL is generally better but requires special consideration on NFS
	db.Exec("PRAGMA journal_mode=WAL;")

	// Use a higher synchronous setting on NFS to prevent corruption
	// FULL is safer on unreliable NFS but slower, NORMAL is a compromise
	db.Exec("PRAGMA synchronous=NORMAL;")

	// Increase page size slightly for NFS to reduce number of I/O operations
	db.Exec("PRAGMA page_size=8192;")

	// Increase cache size to 256MB (-262144 KB) to reduce NFS reads
	db.Exec("PRAGMA cache_size=-262144;")

	// Memory mapping on NFS can be tricky - increase or disable based on testing
	db.Exec("PRAGMA mmap_size=536870912;") // 512MB

	// Keep temp operations in memory to avoid NFS writes
	db.Exec("PRAGMA temp_store=MEMORY;")

	// Larger journal size limit (64MB) for less frequent checkpoints
	db.Exec("PRAGMA journal_size_limit=67108864;")

	// NFS can have high latency - increase timeout for busy conditions
	db.Exec("PRAGMA busy_timeout=10000;") // 10 seconds

	// Exclusive locking mode if possible (if single process access)
	// Comment out if multiple processes need to access the DB
	db.Exec("PRAGMA locking_mode=EXCLUSIVE;")

	// Less frequent but more thorough checkpoints
	db.Exec("PRAGMA wal_autocheckpoint=1000;")

	// Set a reasonable WAL checkpoint timeout
	db.Exec("PRAGMA wal_checkpoint_timeout=30000;")
}
