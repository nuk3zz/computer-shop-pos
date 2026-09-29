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
	if _, err := db.Exec(`
		ALTER TABLE products ADD COLUMN IF NOT EXISTS preorder_enabled BOOLEAN NOT NULL DEFAULT false;
		ALTER TABLE orders ADD COLUMN IF NOT EXISTS fulfillment_type VARCHAR(30) NOT NULL DEFAULT 'in_store';
		ALTER TABLE orders ADD COLUMN IF NOT EXISTS fulfillment_status VARCHAR(30) NOT NULL DEFAULT 'completed';
		ALTER TABLE orders ADD COLUMN IF NOT EXISTS stock_committed BOOLEAN NOT NULL DEFAULT false;
		CREATE INDEX IF NOT EXISTS idx_orders_fulfillment ON orders(order_type, fulfillment_status);
		CREATE TABLE IF NOT EXISTS suppliers (id UUID PRIMARY KEY, name VARCHAR(150) NOT NULL, phone VARCHAR(30), location TEXT, notes TEXT, credit_allowed BOOLEAN NOT NULL DEFAULT false, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE IF NOT EXISTS supplier_purchases (id UUID PRIMARY KEY, supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT, reference_number VARCHAR(100), total_amount NUMERIC(12,2) NOT NULL DEFAULT 0, notes TEXT, purchased_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE IF NOT EXISTS supplier_purchase_items (id UUID PRIMARY KEY, purchase_id UUID NOT NULL REFERENCES supplier_purchases(id) ON DELETE CASCADE, product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT, quantity INTEGER NOT NULL CHECK (quantity > 0), unit_cost NUMERIC(12,2) NOT NULL CHECK (unit_cost >= 0), total_cost NUMERIC(12,2) NOT NULL CHECK (total_cost >= 0));
		CREATE TABLE IF NOT EXISTS supplier_payments (id UUID PRIMARY KEY, supplier_id UUID NOT NULL REFERENCES suppliers(id) ON DELETE RESTRICT, purchase_id UUID REFERENCES supplier_purchases(id) ON DELETE SET NULL, amount NUMERIC(12,2) NOT NULL CHECK (amount > 0), notes TEXT, paid_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		ALTER TABLE supplier_purchases ADD COLUMN IF NOT EXISTS attachment_url TEXT;
		ALTER TABLE supplier_payments ADD COLUMN IF NOT EXISTS attachment_url TEXT;
		CREATE INDEX IF NOT EXISTS idx_supplier_purchases_supplier ON supplier_purchases(supplier_id, purchased_at);
		CREATE INDEX IF NOT EXISTS idx_supplier_payments_supplier ON supplier_payments(supplier_id, paid_at);
	`); err != nil {
		db.Close()
		return nil, fmt.Errorf("apply product order migration: %w", err)
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
		{"products", "preorder_enabled", "BOOLEAN NOT NULL DEFAULT 0"},
		{"orders", "fulfillment_type", "TEXT NOT NULL DEFAULT 'in_store'"},
		{"orders", "fulfillment_status", "TEXT NOT NULL DEFAULT 'completed'"},
		{"orders", "stock_committed", "BOOLEAN NOT NULL DEFAULT 0"},
		{"supplier_purchases", "attachment_url", "TEXT"},
		{"supplier_payments", "attachment_url", "TEXT"},
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
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_orders_fulfillment ON orders(order_type, fulfillment_status)`); err != nil {
		return err
	}
	_, err := db.Exec(`INSERT OR IGNORE INTO schema_migrations (version) VALUES (6)`)
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
