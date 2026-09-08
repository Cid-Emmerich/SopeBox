// Package feed fetches and parses podcast RSS feeds, including the iTunes
// and Podcasting 2.0 namespaces (transcripts, chapters, persons).
package feed

import (
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Cid-Emmerich/SopeBox/internal/art"
)

// Person is a host or guest declared in the feed.
type Person struct {
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
	Img  string `json:"img,omitempty"`
}

// Transcript is a transcript file offered by the feed.
type Transcript struct {
	URL      string `json:"url"`
	Type     string `json:"type"`
	Language string `json:"language,omitempty"`
}

// Episode is one item of a feed.
type Episode struct {
	GUID        string       `json:"guid"`
	Title       string       `json:"title"`
	Description string       `json:"description,omitempty"`
	Link        string       `json:"link,omitempty"`
	Published   time.Time    `json:"published"`
	Duration    float64      `json:"duration,omitempty"` // seconds
	URL         string       `json:"url"`                // enclosure
	Type        string       `json:"type,omitempty"`
	Bytes       int64        `json:"bytes,omitempty"`
	Number      int          `json:"number,omitempty"`
	Season      int          `json:"season,omitempty"`
	Image       string       `json:"image,omitempty"`
	Transcripts []Transcript `json:"transcripts,omitempty"`
	Chapters    string       `json:"chapters,omitempty"`
	Persons     []Person     `json:"persons,omitempty"`
}

// Feed is a parsed podcast.
type Feed struct {
	URL         string    `json:"url"`
	Title       string    `json:"title"`
	Author      string    `json:"author,omitempty"`
	Description string    `json:"description,omitempty"`
	Link        string    `json:"link,omitempty"`
	Image       string    `json:"image,omitempty"`
	Language    string    `json:"language,omitempty"`
	Categories  []string  `json:"categories,omitempty"`
	Persons     []Person  `json:"persons,omitempty"`
	Episodes    []Episode `json:"episodes"`
}

// ---------------------------------------------------------------------------
// XML shapes

type rssDoc struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Title       string        `xml:"title"`
	Link        string        `xml:"link"`
	Description string        `xml:"description"`
	Language    string        `xml:"language"`
	Author      string        `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd author"`
	Summary     string        `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd summary"`
	ItunesImage attrHref      `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Image       rssImage      `xml:"image"`
	Categories  []rssCategory `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd category"`
	Persons     []rssPerson   `xml:"https://podcastindex.org/namespace/1.0 person"`
	Items       []rssItem     `xml:"item"`
}

type rssImage struct {
	URL string `xml:"url"`
}

type attrHref struct {
	Href string `xml:"href,attr"`
}

type rssCategory struct {
	Text string        `xml:"text,attr"`
	Sub  []rssCategory `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd category"`
}

type rssPerson struct {
	Name string `xml:",chardata"`
	Role string `xml:"role,attr"`
	Img  string `xml:"img,attr"`
}

type rssItem struct {
	Title       string          `xml:"title"`
	Link        string          `xml:"link"`
	Description string          `xml:"description"`
	Content     string          `xml:"http://purl.org/rss/1.0/modules/content/ encoded"`
	Summary     string          `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd summary"`
	PubDate     string          `xml:"pubDate"`
	GUID        string          `xml:"guid"`
	Duration    string          `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd duration"`
	Episode     string          `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd episode"`
	Season      string          `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd season"`
	Image       attrHref        `xml:"http://www.itunes.com/dtds/podcast-1.0.dtd image"`
	Enclosure   rssEnclosure    `xml:"enclosure"`
	Transcripts []rssTranscript `xml:"https://podcastindex.org/namespace/1.0 transcript"`
	Chapters    rssChapters     `xml:"https://podcastindex.org/namespace/1.0 chapters"`
	Persons     []rssPerson     `xml:"https://podcastindex.org/namespace/1.0 person"`
}

type rssEnclosure struct {
	URL    string `xml:"url,attr"`
	Type   string `xml:"type,attr"`
	Length string `xml:"length,attr"`
}

type rssTranscript struct {
	URL      string `xml:"url,attr"`
	Type     string `xml:"type,attr"`
	Language string `xml:"language,attr"`
	Rel      string `xml:"rel,attr"`
}

type rssChapters struct {
	URL  string `xml:"url,attr"`
	Type string `xml:"type,attr"`
}

// ---------------------------------------------------------------------------

// Fetch downloads and parses a feed.
func Fetch(u string) (*Feed, error) {
	body, err := art.Get(u)
	if err != nil {
		return nil, err
	}
	f, err := Parse(body)
	if err != nil {
		return nil, err
	}
	f.URL = u
	return f, nil
}

// nsRe finds namespace declarations so alternate spellings of the same
// namespace (the Podcasting 2.0 namespace has had several) can be folded
// into the one the struct tags expect.
var nsRe = regexp.MustCompile(`xmlns:(podcast|itunes)="([^"]*)"`)

func normaliseNamespaces(body string) string {
	return nsRe.ReplaceAllStringFunc(body, func(m string) string {
		sub := nsRe.FindStringSubmatch(m)
		switch sub[1] {
		case "podcast":
			return `xmlns:podcast="https://podcastindex.org/namespace/1.0"`
		case "itunes":
			return `xmlns:itunes="http://www.itunes.com/dtds/podcast-1.0.dtd"`
		}
		return m
	})
}

// Parse parses RSS bytes into a Feed.
func Parse(body []byte) (*Feed, error) {
	var doc rssDoc
	dec := xml.NewDecoder(strings.NewReader(normaliseNamespaces(string(body))))
	dec.Strict = false
	dec.Entity = xml.HTMLEntity
	dec.CharsetReader = charsetReader
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse feed: %w", err)
	}
	ch := doc.Channel
	if ch.Title == "" && len(ch.Items) == 0 {
		return nil, errors.New("not a podcast feed")
	}
	f := &Feed{
		Title:       clean(ch.Title),
		Author:      clean(ch.Author),
		Description: StripHTML(firstNonEmpty(ch.Description, ch.Summary)),
		Link:        strings.TrimSpace(ch.Link),
		Image:       firstNonEmpty(ch.ItunesImage.Href, ch.Image.URL),
		Language:    strings.TrimSpace(ch.Language),
	}
	for _, c := range ch.Categories {
		if c.Text != "" {
			f.Categories = append(f.Categories, html.UnescapeString(c.Text))
		}
	}
	f.Persons = persons(ch.Persons)
	for _, it := range ch.Items {
		if it.Enclosure.URL == "" {
			continue
		}
		e := Episode{
			GUID:        strings.TrimSpace(it.GUID),
			Title:       clean(it.Title),
			Description: StripHTML(firstNonEmpty(it.Content, it.Description, it.Summary)),
			Link:        strings.TrimSpace(it.Link),
			Published:   parseDate(it.PubDate),
			Duration:    ParseDuration(it.Duration),
			URL:         strings.TrimSpace(it.Enclosure.URL),
			Type:        it.Enclosure.Type,
			Image:       it.Image.Href,
			Chapters:    it.Chapters.URL,
			Persons:     persons(it.Persons),
		}
		e.Bytes, _ = strconv.ParseInt(strings.TrimSpace(it.Enclosure.Length), 10, 64)
		e.Number, _ = strconv.Atoi(strings.TrimSpace(it.Episode))
		e.Season, _ = strconv.Atoi(strings.TrimSpace(it.Season))
		if e.GUID == "" {
			e.GUID = e.URL
		}
		for _, t := range it.Transcripts {
			if t.URL != "" {
				e.Transcripts = append(e.Transcripts, Transcript{URL: t.URL, Type: t.Type, Language: t.Language})
			}
		}
		f.Episodes = append(f.Episodes, e)
	}
	return f, nil
}

func persons(ps []rssPerson) []Person {
	var out []Person
	seen := map[string]bool{}
	for _, p := range ps {
		n := clean(p.Name)
		if n == "" || seen[strings.ToLower(n)] {
			continue
		}
		seen[strings.ToLower(n)] = true
		out = append(out, Person{Name: n, Role: p.Role, Img: p.Img})
	}
	return out
}

func firstNonEmpty(ss ...string) string {
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			return s
		}
	}
	return ""
}

func clean(s string) string {
	return strings.Join(strings.Fields(html.UnescapeString(s)), " ")
}

var (
	tagRe   = regexp.MustCompile(`(?s)<[^>]*>`)
	brRe    = regexp.MustCompile(`(?i)<\s*(br|/p|/div|/li|/h[1-6])\s*/?>`)
	spaceRe = regexp.MustCompile(`[ \t]+`)
	nlRe    = regexp.MustCompile(`\n{3,}`)
)

// StripHTML turns show-notes HTML into readable plain text.
func StripHTML(s string) string {
	s = brRe.ReplaceAllString(s, "\n")
	s = tagRe.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = spaceRe.ReplaceAllString(s, " ")
	var lines []string
	for _, l := range strings.Split(s, "\n") {
		lines = append(lines, strings.TrimSpace(l))
	}
	s = strings.Join(lines, "\n")
	s = nlRe.ReplaceAllString(s, "\n\n")
	return strings.TrimSpace(s)
}

var dateLayouts = []string{
	time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822, time.RFC3339,
	"Mon, 2 Jan 2006 15:04:05 -0700", "Mon, 2 Jan 2006 15:04:05 MST",
	"Mon, 02 Jan 2006 15:04:05 -0000", "2 Jan 2006 15:04:05 -0700",
	"2006-01-02 15:04:05", "2006-01-02",
}

func parseDate(s string) time.Time {
	s = strings.TrimSpace(s)
	for _, l := range dateLayouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// ParseDuration understands "3600", "1:00:00", "60:00" and "45:30.5".
func ParseDuration(s string) float64 {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	parts := strings.Split(s, ":")
	total := 0.0
	for _, p := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return 0
		}
		total = total*60 + v
	}
	return total
}
