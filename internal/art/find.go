package art

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// UserAgent identifies SopeBox to podcast hosts and the iTunes API.
const UserAgent = "SopeBox/1.0 (terminal podcast player; +https://github.com/Cid-Emmerich/SopeBox)"

var httpClient = &http.Client{Timeout: 20 * time.Second}

// Get fetches a URL with SopeBox's user agent (max 30 MB).
func Get(u string) ([]byte, error) {
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json, image/*, */*")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 30<<20))
}

// Result is one podcast found online.
type Result struct {
	Title   string
	Author  string
	FeedURL string
	ArtURL  string
	Genre   string
	Count   int
}

// SearchPodcasts queries the iTunes Search API for podcasts by name.
func SearchPodcasts(term string, limit int) ([]Result, error) {
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, errors.New("empty search")
	}
	if limit <= 0 {
		limit = 20
	}
	q := url.Values{"term": {term}, "media": {"podcast"}, "entity": {"podcast"}, "limit": {fmt.Sprint(limit)}}
	body, err := Get("https://itunes.apple.com/search?" + q.Encode())
	if err != nil {
		return nil, err
	}
	var res struct {
		Results []struct {
			ArtistName     string `json:"artistName"`
			CollectionName string `json:"collectionName"`
			FeedURL        string `json:"feedUrl"`
			ArtworkURL600  string `json:"artworkUrl600"`
			ArtworkURL100  string `json:"artworkUrl100"`
			Genre          string `json:"primaryGenreName"`
			TrackCount     int    `json:"trackCount"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, err
	}
	var out []Result
	for _, r := range res.Results {
		art := r.ArtworkURL600
		if art == "" {
			art = strings.Replace(r.ArtworkURL100, "100x100bb", "600x600bb", 1)
		}
		out = append(out, Result{Title: r.CollectionName, Author: r.ArtistName, FeedURL: r.FeedURL, ArtURL: art, Genre: r.Genre, Count: r.TrackCount})
	}
	if len(out) == 0 {
		return nil, errors.New("no results")
	}
	return out, nil
}

// FindIcon searches iTunes for a podcast's artwork by title (and author)
// and returns the raw image bytes plus a description of the match.
func FindIcon(title, author string) ([]byte, string, error) {
	results, err := SearchPodcasts(title, 10)
	if err != nil {
		return nil, "", err
	}
	lt, la := simplify(title), simplify(author)
	best := -1
	for i, r := range results {
		rt, ra := simplify(r.Title), simplify(r.Author)
		titleOK := rt == lt || strings.Contains(rt, lt) || strings.Contains(lt, rt)
		authorOK := la == "" || strings.Contains(ra, la) || strings.Contains(la, ra)
		if titleOK && authorOK {
			best = i
			break
		}
		if titleOK && best == -1 {
			best = i
		}
	}
	if best == -1 {
		best = 0 // iTunes ranks well; take the top hit rather than nothing
	}
	r := results[best]
	if r.ArtURL == "" {
		return nil, "", errors.New("no artwork url")
	}
	data, err := Get(r.ArtURL)
	if err != nil {
		return nil, "", err
	}
	return data, "iTunes: " + r.Title + " – " + r.Author, nil
}

// simplify lower-cases and strips punctuation so small spelling
// differences do not block a match.
func simplify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == ' ', r > 127:
			b.WriteRune(r)
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
