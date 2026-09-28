package database

import (
	"database/sql"
	_ "embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"
)

//go:embed sqlite_schema.sql
var sqliteSchema string

type Config struct {
	Driver   string
	DataDir  string
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

var sqliteConnections sync.Map

func Connect(config Config) (*sql.DB, error) {
	if strings.EqualFold(config.Driver, "sqlite") {
		return connectSQLite(config.DataDir)
	}
	return connectPostgres(config)
}

func connectPostgres(config Config) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Host, config.Port, config.User, config.Password, config.DBName, config.SSLMode,
	)
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database connection: %w", err)
	}
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(5 * time.Minute)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	if _, err := db.Exec(`UPDATE shop_profile SET company_name = 'Universal Repair POS', description = CASE WHEN description = 'Sales and repair management' THEN 'Sales, service, and repair management' ELSE description END, updated_at = CURRENT_TIMESTAMP WHERE company_name = 'Computer Shop POS'`); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply product identity migration: %w", err)
	}
	log.Println("PostgreSQL connection established successfully")
	return db, nil
}

func connectSQLite(dataDir string) (*sql.DB, error) {
	if dataDir == "" {
		dataDir = DefaultDataDir()
	}
	databaseDir := filepath.Join(dataDir, "data")
	if err := os.MkdirAll(databaseDir, 0o700); err != nil {
		return nil, fmt.Errorf("create data directory: %w", err)
	}
	databasePath := filepath.Join(databaseDir, "universal-repair-pos.db")
	legacyDatabasePath := filepath.Join(databaseDir, "computer-shop-pos.db")
	if _, err := os.Stat(databasePath); os.IsNotExist(err) {
		if _, legacyErr := os.Stat(legacyDatabasePath); legacyErr == nil {
			if renameErr := os.Rename(legacyDatabasePath, databasePath); renameErr != nil {
				return nil, fmt.Errorf("migrate legacy SQLite database name: %w", renameErr)
			}
			for _, suffix := range []string{"-wal", "-shm"} {
				_ = os.Rename(legacyDatabasePath+suffix, databasePath+suffix)
			}
		}
	}
	dsn := "file:" + filepath.ToSlash(databasePath) + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_time_format=sqlite"
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open SQLite database: %w", err)
	}
	db.SetMaxOpenConns(5)
	db.SetMaxIdleConns(2)
	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping SQLite database: %w", err)
	}
	if _, err := db.Exec(sqliteSchema); err != nil {
		db.Close()
		return nil, fmt.Errorf("initialize SQLite database: %w", err)
	}
	if err := applySQLiteCompatibilityMigrations(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate SQLite database: %w", err)
	}
	sqliteConnections.Store(db, databasePath)
	log.Printf("SQLite database ready at %s", databasePath)
	return db, nil
}

func applySQLiteCompatibilityMigrations(db *sql.DB) error {
	columns := []struct {
		table, column, definition string
	}{
		{"users", "profile_image_url", "TEXT"},
		{"shop_profile", "description", "TEXT NOT NULL DEFAULT 'Sales and repair management'"},
	}
	for _, migration := range columns {
		exists, err := sqliteColumnExists(db, migration.table, migration.column)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", migration.table, migration.column, migration.definition)); err != nil {
				return err
			}
		}
	}
	if _, err := db.Exec(`UPDATE shop_profile SET company_name = 'Universal Repair POS', description = 'Sales, service, and repair management', updated_at = CURRENT_TIMESTAMP WHERE company_name = 'Computer Shop POS'`); err != nil {
		return err
	}
	_, err := db.Exec(`INSERT OR IGNORE INTO schema_migrations (version) VALUES (3)`)
	return err
}

func sqliteColumnExists(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}

func IsSQLite(db *sql.DB) bool {
	_, ok := sqliteConnections.Load(db)
	return ok
}

func SQLitePath(db *sql.DB) (string, bool) {
	value, ok := sqliteConnections.Load(db)
	if !ok {
		return "", false
	}
	return value.(string), true
}

func DefaultDataDir() string {
	if configured := strings.TrimSpace(os.Getenv("UNIVERSAL_REPAIR_POS_DATA_DIR")); configured != "" {
		return configured
	}
	// Keep the former override working so upgrades never lose sight of existing data.
	if configured := strings.TrimSpace(os.Getenv("COMPUTER_SHOP_DATA_DIR")); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", "Universal Repair POS")
	}
	if runtime.GOOS == "windows" {
		if profile := os.Getenv("USERPROFILE"); profile != "" {
			return preferredDataDir(filepath.Join(profile, "Documents"))
		}
	}
	return preferredDataDir(filepath.Join(home, "Documents"))
}

func preferredDataDir(documentsDir string) string {
	current := filepath.Join(documentsDir, "Universal Repair POS")
	legacy := filepath.Join(documentsDir, "Computer Shop POS")
	if _, err := os.Stat(current); err == nil {
		return current
	}
	if _, err := os.Stat(legacy); err == nil {
		if renameErr := os.Rename(legacy, current); renameErr == nil {
			return current
		}
		return legacy
	}
	return current
}

func IsConnectionError(err error) bool {
	if err == nil {
		return false
	}
	errorString := err.Error()
	for _, fragment := range []string{"connection refused", "no such host", "timeout", "connection reset", "broken pipe", "network is unreachable", "database is locked"} {
		if strings.Contains(errorString, fragment) {
			return true
		}
	}
	return false
}
