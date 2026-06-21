package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr              string
	DatabasePath          string
	MigrationsPath        string
	AllowedOrigin         string
	GoogleSpreadsheetID   string
	GoogleCredentialsFile string
	GoogleSheetName       string
	GoogleDriveFolderID   string
	PhotoStorageDir       string
	SyncInterval          time.Duration
}

func Load() Config {
	loadDotEnv(".env")

	return Config{
		HTTPAddr:              env("HTTP_ADDR", ":8080"),
		DatabasePath:          env("DATABASE_PATH", "data/portaria.db"),
		MigrationsPath:        env("MIGRATIONS_PATH", "migrations"),
		AllowedOrigin:         env("ALLOWED_ORIGIN", "http://localhost:4200"),
		GoogleSpreadsheetID:   envFirst([]string{"GOOGLE_SHEET_ID", "GOOGLE_SPREADSHEET_ID"}, ""),
		GoogleCredentialsFile: env("GOOGLE_CREDENTIALS_FILE", ""),
		GoogleSheetName:       env("GOOGLE_SHEET_NAME", "Entradas"),
		GoogleDriveFolderID:   env("GOOGLE_DRIVE_FOLDER_ID", ""),
		PhotoStorageDir:       env("PHOTO_STORAGE_DIR", "data/photos"),
		SyncInterval:          time.Duration(envInt("SYNC_INTERVAL_SECONDS", 30)) * time.Second,
	}
}

func loadDotEnv(path string) {
	file, err := os.Open(path)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		key, value, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if key == "" || os.Getenv(key) != "" {
			continue
		}
		_ = os.Setenv(key, value)
	}
}

func envFirst(keys []string, fallback string) string {
	for _, key := range keys {
		if value := os.Getenv(key); value != "" {
			return value
		}
	}
	return fallback
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
