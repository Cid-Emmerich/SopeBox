package mentions

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"image"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
)

// Entry is what Wikipedia says about one article.
type Entry struct {
	Title       string    `json:"title"`
	Description string    `json:"description,omitempty"` // "King of England from 1509 to 1547"
	Extract     string    `json:"extract,omitempty"`
	ImageURL    string    `json:"image_url,omitempty"`
	Missing     bool      `json:"missing,omitempty"` // no such article
	Fetched     time.Time `json:"fetched"`
}

// summaryBase is Wikipedia's page summary endpoint (tests point it at a
// local server).
var summaryBase = "https://en.wikipedia.org/api/rest_v1/page/summary/"

func wikiPath(cacheDir, title, ext string) string {
	sum := sha1.Sum([]byte(strings.ToLower(title)))
	return filepath.Join(cacheDir, "wiki", hex.EncodeToString(sum[:8])+ext)
}

// Lookup returns the summary of a Wikipedia article, from the cache when
// possible. A missing article is cached too, as an Entry with Missing set.
func Lookup(cacheDir, title string) (*Entry, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return nil, errors.New("no article title")
	}
	p := wikiPath(cacheDir, title, ".json")
	if raw, err := os.ReadFile(p); err == nil {
		var e Entry
		if json.Unmarshal(raw, &e) == nil {
			return &e, nil
		}
	}
	u := summaryBase + url.PathEscape(strings.ReplaceAll(title, " ", "_"))
	raw, err := art.Get(u)
	var e *Entry
	switch {
	case err != nil && strings.Contains(err.Error(), "HTTP 404"):
		e = &Entry{Title: title, Missing: true}
	case err != nil:
		return nil, err
	default:
		if e, err = parseSummary(raw); err != nil {
			return nil, err
		}
	}
	e.Fetched = time.Now()
	if out, err := json.Marshal(e); err == nil {
		if os.MkdirAll(filepath.Dir(p), 0o755) == nil {
			_ = os.WriteFile(p, out, 0o644)
		}
	}
	return e, nil
}

func parseSummary(raw []byte) (*Entry, error) {
	var s struct {
		Type        string `json:"type"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Extract     string `json:"extract"`
		Thumbnail   struct {
			Source string `json:"source"`
			Width  int    `json:"width"`
		} `json:"thumbnail"`
		Original struct {
			Source string `json:"source"`
			Width  int    `json:"width"`
		} `json:"originalimage"`
	}
	if err := json.Unmarshal(raw, &s); err != nil {
		return nil, err
	}
	e := &Entry{Title: s.Title, Description: s.Description, Extract: s.Extract}
	if s.Type == "disambiguation" {
		// a list of other articles: its picture (if any) is not the thing meant
		return e, nil
	}
	e.ImageURL = pickImage(s.Thumbnail.Source, s.Thumbnail.Width, s.Original.Width)
	return e, nil
}

var thumbWidth = regexp.MustCompile(`/(\d+)px-`)

// imageWidth is the thumbnail width asked for: big enough to look sharp
// in a large terminal, and one of the sizes Wikimedia keeps pre-rendered.
const imageWidth = 500

// pickImage asks for a larger rendering of the summary thumbnail when the
// original is big enough to supply one.
func pickImage(thumb string, thumbW, origW int) string {
	if thumb == "" || thumbW >= imageWidth || origW < imageWidth || !thumbWidth.MatchString(thumb) {
		return thumb
	}
	return thumbWidth.ReplaceAllString(thumb, "/500px-")
}

// Image returns the entry's picture, from the cache when possible.
func Image(cacheDir string, e *Entry) (image.Image, error) {
	if e == nil || e.ImageURL == "" {
		return nil, errors.New("no picture")
	}
	p := wikiPath(cacheDir, e.ImageURL, ".img")
	if raw, err := os.ReadFile(p); err == nil {
		if img, err := art.Decode(raw); err == nil {
			return img, nil
		}
	}
	raw, err := art.Get(e.ImageURL)
	if err != nil {
		return nil, err
	}
	img, err := art.Decode(raw)
	if err != nil {
		return nil, err
	}
	if os.MkdirAll(filepath.Dir(p), 0o755) == nil {
		_ = os.WriteFile(p, raw, 0o644)
	}
	return img, nil
}

// Prefetch looks up and downloads everything a timeline mentions so the
// collage never waits on the network. It stops early if stop is closed.
func Prefetch(cacheDir string, t *Timeline, stop <-chan struct{}) (found, pictures int) {
	if t == nil {
		return
	}
	seen := map[string]bool{}
	for _, m := range t.Mentions {
		if m.Wiki == "" || seen[m.Key()] {
			continue
		}
		seen[m.Key()] = true
		select {
		case <-stop:
			return
		default:
		}
		e, err := Lookup(cacheDir, m.Wiki)
		if err != nil || e.Missing {
			continue
		}
		found++
		if _, err := Image(cacheDir, e); err == nil {
			pictures++
		}
	}
	return
}
