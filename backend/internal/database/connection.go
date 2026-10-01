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
		CREATE TABLE IF NOT EXISTS warranty_claims (id UUID PRIMARY KEY, claim_number VARCHAR(40) UNIQUE NOT NULL, order_id UUID NOT NULL REFERENCES orders(id) ON DELETE RESTRICT, order_item_id UUID NOT NULL REFERENCES order_items(id) ON DELETE RESTRICT, customer_id UUID REFERENCES customers(id) ON DELETE SET NULL, customer_name VARCHAR(150) NOT NULL, customer_phone VARCHAR(30) NOT NULL, product_id UUID NOT NULL REFERENCES products(id) ON DELETE RESTRICT, product_name VARCHAR(200) NOT NULL, serial_number VARCHAR(150), issue_description TEXT NOT NULL, received_condition TEXT, status VARCHAR(30) NOT NULL DEFAULT 'received', resolution VARCHAR(30) NOT NULL DEFAULT 'pending', supplier_status VARCHAR(30) NOT NULL DEFAULT 'not_sent', supplier_recovery_amount NUMERIC(12,2) NOT NULL DEFAULT 0, replacement_source VARCHAR(30) NOT NULL DEFAULT 'none', replacement_product_id UUID REFERENCES products(id) ON DELETE RESTRICT, replacement_stock_committed BOOLEAN NOT NULL DEFAULT false, replacement_cost NUMERIC(12,2) NOT NULL DEFAULT 0, refund_amount NUMERIC(12,2) NOT NULL DEFAULT 0, notes TEXT, received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, completed_at TIMESTAMP, created_by UUID REFERENCES users(id) ON DELETE SET NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		ALTER TABLE warranty_claims ADD COLUMN IF NOT EXISTS supplier_status VARCHAR(30) NOT NULL DEFAULT 'not_sent';
		ALTER TABLE warranty_claims ADD COLUMN IF NOT EXISTS supplier_recovery_amount NUMERIC(12,2) NOT NULL DEFAULT 0;
		ALTER TABLE warranty_claims ADD COLUMN IF NOT EXISTS replacement_source VARCHAR(30) NOT NULL DEFAULT 'none';
		ALTER TABLE warranty_claims ADD COLUMN IF NOT EXISTS replacement_cost NUMERIC(12,2) NOT NULL DEFAULT 0;
		CREATE TABLE IF NOT EXISTS warranty_status_history (id UUID PRIMARY KEY, claim_id UUID NOT NULL REFERENCES warranty_claims(id) ON DELETE CASCADE, previous_status VARCHAR(30), new_status VARCHAR(30) NOT NULL, resolution VARCHAR(30) NOT NULL DEFAULT 'pending', notes TEXT, changed_by UUID REFERENCES users(id) ON DELETE SET NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE INDEX IF NOT EXISTS idx_warranty_claims_status ON warranty_claims(status, received_at);
		CREATE INDEX IF NOT EXISTS idx_warranty_claims_customer ON warranty_claims(customer_id, received_at);
		CREATE INDEX IF NOT EXISTS idx_warranty_history_claim ON warranty_status_history(claim_id, created_at);
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
	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS warranty_claims (id TEXT PRIMARY KEY, claim_number TEXT UNIQUE NOT NULL, order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE RESTRICT, order_item_id TEXT NOT NULL REFERENCES order_items(id) ON DELETE RESTRICT, customer_id TEXT REFERENCES customers(id) ON DELETE SET NULL, customer_name TEXT NOT NULL, customer_phone TEXT NOT NULL, product_id TEXT NOT NULL REFERENCES products(id) ON DELETE RESTRICT, product_name TEXT NOT NULL, serial_number TEXT, issue_description TEXT NOT NULL, received_condition TEXT, status TEXT NOT NULL DEFAULT 'received', resolution TEXT NOT NULL DEFAULT 'pending', supplier_status TEXT NOT NULL DEFAULT 'not_sent', supplier_recovery_amount NUMERIC NOT NULL DEFAULT 0, replacement_source TEXT NOT NULL DEFAULT 'none', replacement_product_id TEXT REFERENCES products(id) ON DELETE RESTRICT, replacement_stock_committed BOOLEAN NOT NULL DEFAULT 0, replacement_cost NUMERIC NOT NULL DEFAULT 0, refund_amount NUMERIC NOT NULL DEFAULT 0, notes TEXT, received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, completed_at TIMESTAMP, created_by TEXT REFERENCES users(id) ON DELETE SET NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP, updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE TABLE IF NOT EXISTS warranty_status_history (id TEXT PRIMARY KEY, claim_id TEXT NOT NULL REFERENCES warranty_claims(id) ON DELETE CASCADE, previous_status TEXT, new_status TEXT NOT NULL, resolution TEXT NOT NULL DEFAULT 'pending', notes TEXT, changed_by TEXT REFERENCES users(id) ON DELETE SET NULL, created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP);
		CREATE INDEX IF NOT EXISTS idx_warranty_claims_status ON warranty_claims(status, received_at);
		CREATE INDEX IF NOT EXISTS idx_warranty_claims_customer ON warranty_claims(customer_id, received_at);
		CREATE INDEX IF NOT EXISTS idx_warranty_history_claim ON warranty_status_history(claim_id, created_at);
	`); err != nil {
		return err
	}
	warrantyColumns := []struct{ column, definition string }{
		{"supplier_status", "TEXT NOT NULL DEFAULT 'not_sent'"},
		{"supplier_recovery_amount", "NUMERIC NOT NULL DEFAULT 0"},
		{"replacement_source", "TEXT NOT NULL DEFAULT 'none'"},
		{"replacement_cost", "NUMERIC NOT NULL DEFAULT 0"},
	}
	for _, migration := range warrantyColumns {
		exists, err := sqliteColumnExists(db, "warranty_claims", migration.column)
		if err != nil {
			return err
		}
		if !exists {
			if _, err := db.Exec(fmt.Sprintf("ALTER TABLE warranty_claims ADD COLUMN %s %s", migration.column, migration.definition)); err != nil {
				return err
			}
		}
	}
	_, err := db.Exec(`INSERT OR IGNORE INTO schema_migrations (version) VALUES (7)`)
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
	if runtime.GOOS == "darwin" {
		path, err := macOSDataDir(home)
		if err != nil {
			log.Fatalf("Could not migrate existing macOS shop data: %v", err)
		}
		return path
	}
	return preferredDataDir(filepath.Join(home, "Documents"))
}

// Check Application Support first so normal launches never touch protected Documents.
// Rename the entire stopped installation directory to retain SQLite sidecars,
// uploads, backups, settings, and authentication identity together.
func macOSDataDir(home string) (string, error) {
	target := filepath.Join(home, "Library", "Application Support", "Universal Repair POS")
	if _, err := os.Stat(target); err == nil {
		return target, nil
	} else if !os.IsNotExist(err) {
		return "", err
	}
	for _, name := range []string{"Universal Repair POS", "Computer Shop POS"} {
		source := filepath.Join(home, "Documents", name)
		if _, err := os.Stat(source); os.IsNotExist(err) {
			continue
		} else if err != nil {
			return "", err
		}
		if err := os.MkdirAll(filepath.Dir(target), 0700); err != nil {
			return "", err
		}
		if err := os.Rename(source, target); err != nil {
			return "", fmt.Errorf("move existing installation to Application Support: %w", err)
		}
		return target, nil
	}
	return target, nil
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
