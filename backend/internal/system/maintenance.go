package system

import (
	"archive/zip"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"pos-backend/internal/database"
)

const backupExtension = ".cspbackup"

type Manager struct {
	db      *sql.DB
	dataDir string
}

type Manifest struct {
	Format     int               `json:"format"`
	CreatedAt  time.Time         `json:"created_at"`
	Kind       string            `json:"kind"`
	AppVersion string            `json:"app_version"`
	Checksums  map[string]string `json:"checksums"`
}

type BackupInfo struct {
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Size      int64     `json:"size"`
	CreatedAt time.Time `json:"created_at"`
}

type restoreMarker struct {
	BackupName string    `json:"backup_name"`
	StagedAt   time.Time `json:"staged_at"`
}

func NewManager(db *sql.DB, dataDir string) *Manager {
	return &Manager{db: db, dataDir: dataDir}
}

func (m *Manager) Supported() bool {
	return database.IsSQLite(m.db)
}

func (m *Manager) DataDir() string { return m.dataDir }

func (m *Manager) BackupDir() string { return filepath.Join(m.dataDir, "backups") }

func (m *Manager) CreateBackup(kind, appVersion string) (BackupInfo, error) {
	if !m.Supported() {
		return BackupInfo{}, errors.New("native backup archives are available in the standalone edition")
	}
	if kind != "automatic" {
		kind = "manual"
	}
	if err := os.MkdirAll(m.BackupDir(), 0o700); err != nil {
		return BackupInfo{}, err
	}
	timestamp := time.Now()
	baseName := fmt.Sprintf("%s-%s", kind, timestamp.Format("20060102-150405"))
	snapshotPath := filepath.Join(m.BackupDir(), baseName+".db.tmp")
	archivePath := filepath.Join(m.BackupDir(), baseName+backupExtension)
	_ = os.Remove(snapshotPath)
	if _, err := m.db.Exec(`VACUUM INTO $1`, snapshotPath); err != nil {
		return BackupInfo{}, fmt.Errorf("create consistent database snapshot: %w", err)
	}
	defer os.Remove(snapshotPath)

	manifest := Manifest{Format: 1, CreatedAt: timestamp, Kind: kind, AppVersion: appVersion, Checksums: map[string]string{}}
	archive, err := os.OpenFile(archivePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return BackupInfo{}, err
	}
	zipWriter := zip.NewWriter(archive)
	writeErr := m.addFile(zipWriter, snapshotPath, "data/computer-shop-pos.db", manifest.Checksums)
	if writeErr == nil {
		uploadRoot := filepath.Join(m.dataDir, "uploads")
		writeErr = filepath.WalkDir(uploadRoot, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			relative, err := filepath.Rel(uploadRoot, path)
			if err != nil {
				return err
			}
			return m.addFile(zipWriter, path, filepath.ToSlash(filepath.Join("uploads", relative)), manifest.Checksums)
		})
		if os.IsNotExist(writeErr) {
			writeErr = nil
		}
	}
	if writeErr == nil {
		manifestWriter, err := zipWriter.Create("manifest.json")
		if err != nil {
			writeErr = err
		} else {
			writeErr = json.NewEncoder(manifestWriter).Encode(manifest)
		}
	}
	closeZipErr := zipWriter.Close()
	closeFileErr := archive.Close()
	if writeErr != nil || closeZipErr != nil || closeFileErr != nil {
		_ = os.Remove(archivePath)
		return BackupInfo{}, errors.Join(writeErr, closeZipErr, closeFileErr)
	}
	info, err := os.Stat(archivePath)
	if err != nil {
		return BackupInfo{}, err
	}
	if kind == "automatic" {
		_ = m.pruneAutomatic(30)
	}
	return BackupInfo{Name: info.Name(), Kind: kind, Size: info.Size(), CreatedAt: info.ModTime()}, nil
}

func (m *Manager) addFile(writer *zip.Writer, source, archiveName string, checksums map[string]string) error {
	input, err := os.Open(source)
	if err != nil {
		return err
	}
	defer input.Close()
	output, err := writer.Create(archiveName)
	if err != nil {
		return err
	}
	hash := sha256.New()
	if _, err := io.Copy(io.MultiWriter(output, hash), input); err != nil {
		return err
	}
	checksums[archiveName] = hex.EncodeToString(hash.Sum(nil))
	return nil
}

func (m *Manager) ListBackups() ([]BackupInfo, error) {
	entries, err := os.ReadDir(m.BackupDir())
	if os.IsNotExist(err) {
		return []BackupInfo{}, nil
	}
	if err != nil {
		return nil, err
	}
	backups := make([]BackupInfo, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), backupExtension) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return nil, err
		}
		kind := "manual"
		if strings.HasPrefix(entry.Name(), "automatic-") {
			kind = "automatic"
		}
		backups = append(backups, BackupInfo{Name: entry.Name(), Kind: kind, Size: info.Size(), CreatedAt: info.ModTime()})
	}
	sort.Slice(backups, func(i, j int) bool { return backups[i].CreatedAt.After(backups[j].CreatedAt) })
	return backups, nil
}

func (m *Manager) BackupPath(name string) (string, error) {
	if filepath.Base(name) != name || !strings.HasSuffix(name, backupExtension) {
		return "", errors.New("invalid backup name")
	}
	path := filepath.Join(m.BackupDir(), name)
	if _, err := os.Stat(path); err != nil {
		return "", err
	}
	return path, nil
}

func (m *Manager) SaveUploadedBackup(source io.Reader) (BackupInfo, error) {
	if err := os.MkdirAll(m.BackupDir(), 0o700); err != nil {
		return BackupInfo{}, err
	}
	name := "imported-" + time.Now().Format("20060102-150405") + backupExtension
	path := filepath.Join(m.BackupDir(), name)
	output, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return BackupInfo{}, err
	}
	limited := io.LimitReader(source, 4<<30)
	_, copyErr := io.Copy(output, limited)
	closeErr := output.Close()
	if copyErr != nil || closeErr != nil {
		_ = os.Remove(path)
		return BackupInfo{}, errors.Join(copyErr, closeErr)
	}
	if err := ValidateBackup(path); err != nil {
		_ = os.Remove(path)
		return BackupInfo{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return BackupInfo{}, err
	}
	return BackupInfo{Name: name, Kind: "imported", Size: info.Size(), CreatedAt: info.ModTime()}, nil
}

func (m *Manager) StageRestore(name, appVersion string) error {
	path, err := m.BackupPath(name)
	if err != nil {
		return err
	}
	if err := ValidateBackup(path); err != nil {
		return err
	}
	if _, err := m.CreateBackup("manual", appVersion); err != nil {
		return fmt.Errorf("create pre-restore safety backup: %w", err)
	}
	stageDir := filepath.Join(m.dataDir, "data", "restore-staging")
	_ = os.RemoveAll(stageDir)
	if err := os.MkdirAll(stageDir, 0o700); err != nil {
		return err
	}
	if err := extractBackup(path, stageDir); err != nil {
		_ = os.RemoveAll(stageDir)
		return err
	}
	configDir := filepath.Join(m.dataDir, "config")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		return err
	}
	marker, _ := json.Marshal(restoreMarker{BackupName: name, StagedAt: time.Now()})
	return os.WriteFile(filepath.Join(configDir, "restore-pending.json"), marker, 0o600)
}

func ValidateBackup(path string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return fmt.Errorf("open backup archive: %w", err)
	}
	defer reader.Close()
	var manifest Manifest
	foundDatabase := false
	for _, file := range reader.File {
		if file.Name != "manifest.json" {
			continue
		}
		input, err := file.Open()
		if err != nil {
			return err
		}
		err = json.NewDecoder(input).Decode(&manifest)
		_ = input.Close()
		if err != nil {
			return fmt.Errorf("read backup manifest: %w", err)
		}
	}
	if manifest.Format != 1 || manifest.Checksums == nil {
		return errors.New("unsupported or missing backup manifest")
	}
	for _, file := range reader.File {
		expected, tracked := manifest.Checksums[file.Name]
		if !tracked {
			continue
		}
		input, err := file.Open()
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, copyErr := io.Copy(hash, input)
		_ = input.Close()
		if copyErr != nil || hex.EncodeToString(hash.Sum(nil)) != expected {
			return fmt.Errorf("backup checksum failed for %s", file.Name)
		}
		if file.Name == "data/computer-shop-pos.db" {
			foundDatabase = true
		}
	}
	if !foundDatabase {
		return errors.New("backup does not contain a database")
	}
	return nil
}

func extractBackup(path, target string) error {
	reader, err := zip.OpenReader(path)
	if err != nil {
		return err
	}
	defer reader.Close()
	for _, file := range reader.File {
		if file.Name == "manifest.json" {
			continue
		}
		clean := filepath.Clean(filepath.FromSlash(file.Name))
		if clean == "." || strings.HasPrefix(clean, ".."+string(os.PathSeparator)) || filepath.IsAbs(clean) {
			return errors.New("backup contains an unsafe path")
		}
		destination := filepath.Join(target, clean)
		if file.FileInfo().IsDir() {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(destination), 0o700); err != nil {
			return err
		}
		input, err := file.Open()
		if err != nil {
			return err
		}
		output, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			_ = input.Close()
			return err
		}
		_, copyErr := io.Copy(output, input)
		closeOutputErr := output.Close()
		closeInputErr := input.Close()
		if err := errors.Join(copyErr, closeOutputErr, closeInputErr); err != nil {
			return err
		}
	}
	return nil
}

func ApplyPendingRestore(dataDir string) (bool, error) {
	markerPath := filepath.Join(dataDir, "config", "restore-pending.json")
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	stageDir := filepath.Join(dataDir, "data", "restore-staging")
	stagedDatabase := filepath.Join(stageDir, "data", "computer-shop-pos.db")
	if _, err := os.Stat(stagedDatabase); err != nil {
		return false, fmt.Errorf("staged restore database is missing: %w", err)
	}
	databasePath := filepath.Join(dataDir, "data", "computer-shop-pos.db")
	suffix := time.Now().Format("20060102-150405")
	if _, err := os.Stat(databasePath); err == nil {
		if err := os.Rename(databasePath, databasePath+".before-restore-"+suffix); err != nil {
			return false, err
		}
	}
	_ = os.Remove(databasePath + "-wal")
	_ = os.Remove(databasePath + "-shm")
	if err := os.Rename(stagedDatabase, databasePath); err != nil {
		return false, err
	}
	stagedUploads := filepath.Join(stageDir, "uploads")
	uploadsPath := filepath.Join(dataDir, "uploads")
	if _, err := os.Stat(stagedUploads); err == nil {
		if _, err := os.Stat(uploadsPath); err == nil {
			if err := os.Rename(uploadsPath, uploadsPath+".before-restore-"+suffix); err != nil {
				return false, err
			}
		}
		if err := os.Rename(stagedUploads, uploadsPath); err != nil {
			return false, err
		}
	}
	if err := os.Remove(markerPath); err != nil {
		return false, err
	}
	_ = os.RemoveAll(stageDir)
	return true, nil
}

func (m *Manager) StartAutomaticBackups(ctx context.Context, appVersion string) {
	if !m.Supported() {
		return
	}
	run := func() {
		var enabled bool
		var backupTime string
		if err := m.db.QueryRow(`SELECT auto_backup, backup_time FROM shop_profile WHERE id = 1`).Scan(&enabled, &backupTime); err != nil || !enabled {
			return
		}
		hour, minute := 2, 30
		_, _ = fmt.Sscanf(backupTime, "%d:%d", &hour, &minute)
		now := time.Now()
		if now.Before(time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())) {
			return
		}
		prefix := "automatic-" + now.Format("20060102")
		backups, _ := m.ListBackups()
		for _, backup := range backups {
			if strings.HasPrefix(backup.Name, prefix) {
				return
			}
		}
		if _, err := m.CreateBackup("automatic", appVersion); err != nil {
			log.Printf("Automatic backup failed: %v", err)
		}
	}
	run()
	ticker := time.NewTicker(15 * time.Minute)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				run()
			}
		}
	}()
}

func (m *Manager) pruneAutomatic(keep int) error {
	backups, err := m.ListBackups()
	if err != nil {
		return err
	}
	count := 0
	for _, backup := range backups {
		if backup.Kind != "automatic" {
			continue
		}
		count++
		if count > keep {
			_ = os.Remove(filepath.Join(m.BackupDir(), backup.Name))
		}
	}
	return nil
}
