package photos

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type Store struct {
	dir string
}

func NewStore(dir string) *Store {
	return &Store{dir: strings.TrimSpace(dir)}
}

func (s *Store) Dir() string {
	return s.dir
}

func (s *Store) SaveDataURL(ctx context.Context, category string, dataURL string) (string, error) {
	dataURL = strings.TrimSpace(dataURL)
	if dataURL == "" || !strings.HasPrefix(dataURL, "data:image/") || s.dir == "" {
		return dataURL, nil
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}

	contentType, encoded, found := strings.Cut(dataURL, ",")
	if !found {
		return "", fmt.Errorf("invalid photo data url")
	}
	contentType = strings.TrimPrefix(contentType, "data:")
	if mediaType, _, found := strings.Cut(contentType, ";"); found {
		contentType = mediaType
	}

	exts, err := mime.ExtensionsByType(contentType)
	if err != nil || len(exts) == 0 {
		return "", fmt.Errorf("unsupported photo type: %s", contentType)
	}

	category = sanitize(category)
	if category == "" {
		category = "foto"
	}

	if err := os.MkdirAll(filepath.Join(s.dir, category), 0o755); err != nil {
		return "", fmt.Errorf("create photo dir: %w", err)
	}

	name := fmt.Sprintf("%s-%s%s", time.Now().Format("20060102-150405.000"), randomSuffix(), exts[0])
	targetPath := filepath.Join(s.dir, category, name)
	file, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		return "", fmt.Errorf("create photo file: %w", err)
	}
	defer file.Close()

	decoder := base64.NewDecoder(base64.StdEncoding, strings.NewReader(encoded))
	if _, err := io.Copy(file, decoder); err != nil {
		_ = os.Remove(targetPath)
		return "", fmt.Errorf("write photo file: %w", err)
	}

	return "/api/photos/" + category + "/" + name, nil
}

func sanitize(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = regexp.MustCompile(`[^a-z0-9._-]+`).ReplaceAllString(value, "-")
	return strings.Trim(value, "-_.")
}

func randomSuffix() string {
	var buf [4]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "00000000"
	}
	return fmt.Sprintf("%x", buf[:])
}
