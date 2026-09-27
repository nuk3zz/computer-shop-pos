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
	databasePath := filepath.Join(databaseDir, "computer-shop-pos.db")
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
	sqliteConnections.Store(db, databasePath)
	log.Printf("SQLite database ready at %s", databasePath)
	return db, nil
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
	if configured := strings.TrimSpace(os.Getenv("COMPUTER_SHOP_DATA_DIR")); configured != "" {
		return configured
	}
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		return filepath.Join(".", "Computer Shop POS")
	}
	if runtime.GOOS == "windows" {
		if profile := os.Getenv("USERPROFILE"); profile != "" {
			return filepath.Join(profile, "Documents", "Computer Shop POS")
		}
	}
	return filepath.Join(home, "Documents", "Computer Shop POS")
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
