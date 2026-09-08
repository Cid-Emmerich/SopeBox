package feed

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
)

// CasteroDB returns the default castero database path if it exists.
func CasteroDB() string {
	home, _ := os.UserHomeDir()
	cands := []string{filepath.Join(home, ".local", "share", "castero", "castero.db")}
	if x := os.Getenv("XDG_DATA_HOME"); x != "" {
		cands = append([]string{filepath.Join(x, "castero", "castero.db")}, cands...)
	}
	for _, c := range cands {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return ""
}

// ImportCastero reads subscriptions out of a castero SQLite database using
// the sqlite3 command line tool (present on macOS and most Linux systems).
func ImportCastero(db string) ([]Sub, error) {
	if db == "" {
		db = CasteroDB()
	}
	if db == "" {
		return nil, errors.New("castero database not found")
	}
	sq, err := exec.LookPath("sqlite3")
	if err != nil {
		return nil, errors.New("sqlite3 is needed to read the castero database")
	}
	out, err := exec.Command(sq, "-json", db, "select key, title from feed").Output()
	if err != nil {
		return nil, err
	}
	var rows []struct {
		Key   string `json:"key"`
		Title string `json:"title"`
	}
	if len(out) > 0 {
		if err := json.Unmarshal(out, &rows); err != nil {
			return nil, err
		}
	}
	var subs []Sub
	for _, r := range rows {
		if r.Key != "" {
			subs = append(subs, Sub{Title: r.Title, URL: r.Key})
		}
	}
	return subs, nil
}
