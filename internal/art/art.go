// Package art loads, finds and renders podcast artwork ("icons").
package art

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"path/filepath"

	_ "golang.org/x/image/bmp"
	_ "golang.org/x/image/webp"
)

// Icon is a decoded artwork with where it came from.
type Icon struct {
	Image  image.Image
	Path   string
	Source string
}

// Decode decodes any supported image format.
func Decode(raw []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(raw))
	return img, err
}

// CachePath returns where the icon for a key (feed URL) lives on disk.
func CachePath(cacheDir, key string) string {
	sum := sha1.Sum([]byte(key))
	return filepath.Join(cacheDir, "icons", hex.EncodeToString(sum[:8])+".img")
}

// LoadCached returns the cached icon for a key, or nil.
func LoadCached(cacheDir, key string) *Icon {
	p := CachePath(cacheDir, key)
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	img, err := Decode(raw)
	if err != nil {
		return nil
	}
	return &Icon{Image: img, Path: p, Source: "cache"}
}

// Store writes raw image bytes into the cache for a key.
func Store(cacheDir, key string, raw []byte) (string, error) {
	p := CachePath(cacheDir, key)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", err
	}
	return p, os.WriteFile(p, raw, 0o644)
}
