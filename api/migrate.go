package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/norskhelsenett/chase/scheduler"
	"github.com/norskhelsenett/chase/security"
	"github.com/norskhelsenett/chase/servers"
	"github.com/norskhelsenett/chase/session"
	"github.com/norskhelsenett/chase/types"
	"github.com/norskhelsenett/chase/webpush"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

// dataMigration tracks one-time data imports so this step is idempotent.
type dataMigration struct {
	Name       string `gorm:"primaryKey;type:varchar(64)"`
	AppliedAt  time.Time
	SourcePath string
	RowCounts  string `gorm:"type:text"`
}

const sqliteImportName = "sqlite_to_postgres"

// legacySQLiteNames lists every database filename chase has shipped with;
// historical deployments may have any of them on disk.
var legacySQLiteNames = []string{"chase.db", "fit.db"}

// migrateFromSQLite copies all rows from a legacy SQLite database into the
// current Postgres connection in a single transaction. It is a no-op when
// the source is missing or the import has already been applied, so the same
// binary works in fresh installs, devcontainer runs, and one-shot cutovers.
//
// Path resolution ($MIGRATE_FROM_SQLITE):
//   - existing file → use it directly
//   - existing directory → first matching legacySQLiteNames entry
//   - non-existent path → check legacySQLiteNames in its parent directory
//
// Idempotency:
//   - A row in data_migrations marks completion; re-runs short-circuit.
//
// Atomicity:
//   - All copies + the marker insert + serial-sequence advances run in one
//     dst.Transaction; any error rolls the whole import back.
func migrateFromSQLite(dst *gorm.DB) error {
	hint := strings.TrimSpace(os.Getenv("MIGRATE_FROM_SQLITE"))
	if hint == "" {
		return nil
	}
	path := resolveLegacyDB(hint)
	if path == "" {
		log.Printf("MIGRATE_FROM_SQLITE=%s — no legacy SQLite file found, skipping import", hint)
		return nil
	}

	// Ensure every destination table exists before we touch it. Some
	// AutoMigrate calls happen later in main.go (e.g. scheduler.New) so we
	// can't rely on call ordering.
	if err := dst.AutoMigrate(
		&dataMigration{},
		&types.User{},
		&servers.Server{}, &servers.PingDetail{}, &servers.PingResult{},
		&servers.PingHourlySummary{}, &servers.PingDailySummary{}, &servers.GeoCache{},
		&security.BatchJobStore{}, &security.BatchResultStore{},
		&security.SecurityReportRecord{}, &security.Screenshot{},
		&webpush.VAPIDKeys{}, &webpush.PushSubscription{},
		&webpush.NotificationPreference{}, &webpush.NotificationLog{},
		&scheduler.JobRunRecord{},
		&session.Session{},
	); err != nil {
		return fmt.Errorf("ensure destination tables: %w", err)
	}

	var existing dataMigration
	if err := dst.Where("name = ?", sqliteImportName).First(&existing).Error; err == nil {
		log.Printf("SQLite import already applied at %s — skipping", existing.AppliedAt.Format(time.RFC3339))
		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("check existing migration: %w", err)
	}

	src, err := gorm.Open(sqlite.Open(path+"?mode=ro&_pragma=journal_mode(off)"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return fmt.Errorf("open sqlite source %s: %w", path, err)
	}
	if sqlDB, err := src.DB(); err == nil {
		defer sqlDB.Close()
	}

	log.Printf("Importing data from SQLite %s into Postgres...", path)
	counts := map[string]int64{}

	err = dst.Transaction(func(tx *gorm.DB) error {
		// Order respects FK dependencies: parents before children.
		copiers := []func() error{
			func() error { return copyTable[types.User](src, tx, counts) },
			func() error { return copyTable[servers.Server](src, tx, counts) },
			func() error { return copyTable[servers.PingDetail](src, tx, counts) },
			func() error { return copyTable[servers.PingResult](src, tx, counts) },
			func() error { return copyTable[servers.PingHourlySummary](src, tx, counts) },
			func() error { return copyTable[servers.PingDailySummary](src, tx, counts) },
			func() error { return copyTable[servers.GeoCache](src, tx, counts) },
			func() error { return copyTable[security.BatchJobStore](src, tx, counts) },
			func() error { return copyTable[security.BatchResultStore](src, tx, counts) },
			func() error { return copyTable[security.SecurityReportRecord](src, tx, counts) },
			func() error { return copyTable[security.Screenshot](src, tx, counts) },
			func() error { return copyTable[webpush.VAPIDKeys](src, tx, counts) },
			func() error { return copyTable[webpush.PushSubscription](src, tx, counts) },
			func() error { return copyTable[webpush.NotificationPreference](src, tx, counts) },
			func() error { return copyTable[webpush.NotificationLog](src, tx, counts) },
			func() error { return copyTable[scheduler.JobRunRecord](src, tx, counts) },
			func() error { return copyTable[session.Session](src, tx, counts) },
		}
		for _, fn := range copiers {
			if err := fn(); err != nil {
				return err
			}
		}

		// Bulk-insert preserved IDs but Postgres serial sequences still point
		// at 1 — the next INSERT without an explicit id would collide. Advance
		// every id sequence past the current MAX(id) of its table.
		if err := advanceSerialSequences(tx); err != nil {
			return fmt.Errorf("advance sequences: %w", err)
		}

		return tx.Create(&dataMigration{
			Name:       sqliteImportName,
			AppliedAt:  time.Now(),
			SourcePath: path,
			RowCounts:  formatCounts(counts),
		}).Error
	})
	if err != nil {
		return fmt.Errorf("sqlite import: %w", err)
	}

	log.Printf("SQLite import complete — %s", formatCounts(counts))
	return nil
}

// copyTable reads every row of T from src (including soft-deleted) and
// bulk-inserts them into the dst transaction. Empty source tables are fine.
func copyTable[T any](src, tx *gorm.DB, counts map[string]int64) error {
	var rows []T
	// Unscoped() so soft-deleted rows are preserved.
	if err := src.Unscoped().Find(&rows).Error; err != nil {
		// Source may not have this table at all (older SQLite schemas) — treat
		// missing table as empty.
		if isMissingTable(err) {
			return nil
		}
		return fmt.Errorf("read %T: %w", *new(T), err)
	}
	if len(rows) == 0 {
		return nil
	}
	// UpdateAll on primary-key conflict: any rows inits (e.g. webpush VAPID
	// keys generated before migrateFromSQLite ran) get replaced by the SQLite
	// source, which is the canonical pre-cutover state.
	if err := tx.Session(&gorm.Session{SkipHooks: true}).
		Clauses(clause.OnConflict{UpdateAll: true}).
		CreateInBatches(rows, 200).Error; err != nil {
		return fmt.Errorf("write %T (%d rows): %w", *new(T), len(rows), err)
	}
	counts[fmt.Sprintf("%T", *new(T))] = int64(len(rows))
	return nil
}

func isMissingTable(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "no such table")
}

// advanceSerialSequences finds every owned id sequence in the public schema
// and sets it to MAX(id) of the owning table, so subsequent inserts allocate
// fresh ids without colliding with imported rows.
func advanceSerialSequences(tx *gorm.DB) error {
	const sql = `
DO $$
DECLARE r record;
BEGIN
  FOR r IN
    SELECT s.relname AS seq, t.relname AS tbl, a.attname AS col
    FROM pg_class s
    JOIN pg_depend d ON d.objid = s.oid AND d.deptype = 'a'
    JOIN pg_class t ON t.oid = d.refobjid
    JOIN pg_attribute a ON a.attrelid = t.oid AND a.attnum = d.refobjsubid
    JOIN pg_namespace n ON n.oid = t.relnamespace
    WHERE s.relkind = 'S' AND n.nspname = 'public'
  LOOP
    EXECUTE format(
      'SELECT setval(%L, COALESCE((SELECT MAX(%I) FROM %I), 0) + 1, false)',
      r.seq, r.col, r.tbl);
  END LOOP;
END $$;`
	return tx.Exec(sql).Error
}

// resolveLegacyDB returns an existing path to a legacy chase SQLite database,
// or "" if none was found. See migrateFromSQLite for resolution rules.
func resolveLegacyDB(hint string) string {
	info, err := os.Stat(hint)
	if err == nil {
		if info.IsDir() {
			return firstExistingIn(hint)
		}
		return hint
	}
	// Path doesn't exist as-is; treat it as a candidate filename and look
	// for a sibling with a known historical name.
	if dir := filepath.Dir(hint); dir != "" && dir != "." {
		if got := firstExistingIn(dir); got != "" {
			return got
		}
	}
	return ""
}

// firstExistingIn picks a SQLite database from dir. Preferred filenames
// (legacySQLiteNames) win; otherwise it falls back to any *.db file so
// deployments that customized the filename still migrate cleanly.
func firstExistingIn(dir string) string {
	for _, name := range legacySQLiteNames {
		p := filepath.Join(dir, name)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.db"))
	sort.Strings(matches)
	for _, p := range matches {
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p
		}
	}
	return ""
}

func formatCounts(counts map[string]int64) string {
	if len(counts) == 0 {
		return "no rows imported"
	}
	keys := make([]string, 0, len(counts))
	for k := range counts {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for i, k := range keys {
		if i > 0 {
			b.WriteString(", ")
		}
		short := k
		if idx := strings.LastIndex(k, "."); idx >= 0 {
			short = k[idx+1:]
		}
		fmt.Fprintf(&b, "%s=%d", short, counts[k])
	}
	return b.String()
}
