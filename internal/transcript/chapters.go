package transcript

import (
	"encoding/json"
	"errors"
	"sort"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
)

// Chapter is a Podcasting 2.0 chapter marker.
type Chapter struct {
	Start float64 `json:"startTime"`
	Title string  `json:"title"`
	URL   string  `json:"url,omitempty"`
}

// FetchChapters downloads a podcast:chapters JSON file.
func FetchChapters(u string) ([]Chapter, error) {
	if u == "" {
		return nil, errors.New("no chapters")
	}
	raw, err := art.Get(u)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Chapters []Chapter `json:"chapters"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return nil, err
	}
	sort.SliceStable(doc.Chapters, func(i, j int) bool { return doc.Chapters[i].Start < doc.Chapters[j].Start })
	return doc.Chapters, nil
}

// ChapterAt returns the index of the chapter containing pos (-1 if none).
func ChapterAt(ch []Chapter, pos float64) int {
	i := sort.Search(len(ch), func(i int) bool { return ch[i].Start > pos }) - 1
	return i
}
