package main

import (
	"context"
	"crypto/rand"
	"database/sql"
	"embed"
	"encoding/hex"
	"flag"
	"io"
	"io/fs"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pos-backend/internal/api"
	"pos-backend/internal/database"
	"pos-backend/internal/middleware"
	possystem "pos-backend/internal/system"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

var (
	version   = "dev"
	commit    = "unknown"
	buildDate = "unknown"
)

//go:embed web/*
var embeddedWeb embed.FS

func main() {
	nativeFlag := flag.Bool("native", false, "run the standalone SQLite edition")
	dataDirFlag := flag.String("data-dir", "", "standalone data directory")
	portFlag := flag.String("port", "", "HTTP port")
	versionFlag := flag.Bool("version", false, "print version and exit")
	flag.Parse()
	if *versionFlag {
		log.Printf("Computer Shop POS %s (%s, %s)", version, commit, buildDate)
		return
	}

	_ = godotenv.Load()
	native := *nativeFlag || strings.EqualFold(os.Getenv("DB_DRIVER"), "sqlite") || os.Getenv("COMPUTER_SHOP_NATIVE") == "1"
	dataDir := *dataDirFlag
	if dataDir == "" {
		dataDir = database.DefaultDataDir()
	}
	if native {
		logFile, err := configureNativeLogging(dataDir)
		if err != nil {
			log.Printf("Could not open native log file: %v", err)
		} else {
			defer logFile.Close()
		}
		if restored, err := possystem.ApplyPendingRestore(dataDir); err != nil {
			log.Fatalf("Failed to apply pending restore: %v", err)
		} else if restored {
			log.Println("Verified backup restore applied successfully")
		}
	}

	dbConfig := database.Config{
		Driver:   getEnv("DB_DRIVER", "postgres"),
		DataDir:  dataDir,
		Host:     getEnv("DB_HOST", "postgres"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", "postgres123"),
		DBName:   getEnv("DB_NAME", "pos_system"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
	if native {
		dbConfig.Driver = "sqlite"
	}

	db, err := database.Connect(dbConfig)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if native {
		secret, err := loadOrCreateSecret(filepath.Join(dataDir, "config", "jwt-secret"))
		if err != nil {
			log.Fatalf("Failed to initialize installation secret: %v", err)
		}
		middleware.SetJWTSecret(secret)
	}

	gin.SetMode(getEnv("GIN_MODE", "release"))
	router := gin.New()
	router.MaxMultipartMemory = 11 << 20

	uploadDir := getEnv("UPLOAD_DIR", "uploads")
	if native {
		uploadDir = filepath.Join(dataDir, "uploads")
	}
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}
	router.Static("/uploads", uploadDir)

	router.Use(gin.Logger(), gin.Recovery())
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:5173"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Content-Length", "Accept-Encoding", "X-CSRF-Token", "Authorization", "accept", "origin", "Cache-Control", "X-Requested-With"},
		AllowCredentials: true,
	}))
	if native {
		router.Use(networkModeMiddleware(db))
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "version": version, "native": native})
	})

	apiRoutes := router.Group("/api/v1")
	api.SetupRoutes(apiRoutes, db, middleware.AuthMiddleware(), uploadDir, dataDir, version)
	if native {
		possystem.NewManager(db, dataDir).StartAutomaticBackups(context.Background(), version)
	}

	if native {
		registerEmbeddedWeb(router)
	}

	port := *portFlag
	if port == "" {
		if native {
			port = getEnv("PORT", "3000")
		} else {
			port = getEnv("PORT", "8080")
		}
	}
	log.Printf("Computer Shop POS %s starting at http://localhost:%s", version, port)
	if native {
		for _, address := range lanAddresses() {
			log.Printf("LAN address: http://%s:%s", address, port)
		}
	}
	if err := router.Run("0.0.0.0:" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func configureNativeLogging(dataDir string) (*os.File, error) {
	logDir := filepath.Join(dataDir, "logs")
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return nil, err
	}
	logFile, err := os.OpenFile(filepath.Join(logDir, "server.log"), os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, err
	}
	log.SetOutput(io.MultiWriter(logFile, os.Stdout))
	return logFile, nil
}

func registerEmbeddedWeb(router *gin.Engine) {
	webRoot, err := fs.Sub(embeddedWeb, "web")
	if err != nil {
		log.Fatalf("Prepare embedded web UI: %v", err)
	}
	indexHTML, err := fs.ReadFile(webRoot, "index.html")
	if err != nil {
		log.Fatalf("Read embedded web UI: %v", err)
	}
	router.NoRoute(func(c *gin.Context) {
		if c.Request.Method != http.MethodGet || strings.HasPrefix(c.Request.URL.Path, "/api/") {
			c.Status(http.StatusNotFound)
			return
		}
		requested := strings.TrimPrefix(filepath.ToSlash(c.Request.URL.Path), "/")
		if requested != "" {
			if file, openErr := webRoot.Open(requested); openErr == nil {
				_ = file.Close()
				c.FileFromFS(requested, http.FS(webRoot))
				return
			}
		}
		c.Data(http.StatusOK, "text/html; charset=utf-8", indexHTML)
	})
}

func networkModeMiddleware(db interface {
	QueryRow(string, ...interface{}) *sql.Row
}) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := directRemoteIP(c.Request.RemoteAddr)
		if ip == nil || ip.IsLoopback() {
			c.Next()
			return
		}
		var setupCompleted bool
		if ip.IsPrivate() && db.QueryRow(`SELECT setup_completed FROM shop_profile WHERE id = 1`).Scan(&setupCompleted) == nil && !setupCompleted {
			c.Next()
			return
		}
		var mode string
		if err := db.QueryRow(`SELECT network_mode FROM shop_profile WHERE id = 1`).Scan(&mode); err != nil || mode != "lan" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "message": "This installation is currently limited to this computer"})
			c.Abort()
			return
		}
		c.Next()
	}
}

func directRemoteIP(remoteAddress string) net.IP {
	host, _, err := net.SplitHostPort(remoteAddress)
	if err != nil {
		host = remoteAddress
	}
	return net.ParseIP(strings.Trim(host, "[]"))
}

func loadOrCreateSecret(path string) (string, error) {
	if stored, err := os.ReadFile(path); err == nil && strings.TrimSpace(string(stored)) != "" {
		return strings.TrimSpace(string(stored)), nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return "", err
	}
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}
	secret := hex.EncodeToString(random)
	if err := os.WriteFile(path, []byte(secret), 0o600); err != nil {
		return "", err
	}
	return secret, nil
}

func lanAddresses() []string {
	var addresses []string
	interfaces, _ := net.Interfaces()
	for _, networkInterface := range interfaces {
		interfaceAddresses, _ := networkInterface.Addrs()
		for _, address := range interfaceAddresses {
			ip, _, err := net.ParseCIDR(address.String())
			if err == nil && ip.To4() != nil && !ip.IsLoopback() {
				addresses = append(addresses, ip.String())
			}
		}
	}
	return addresses
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
