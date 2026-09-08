package feed

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

type opmlDoc struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    opmlHead `xml:"head"`
	Body    struct {
		Outlines []opmlOutline `xml:"outline"`
	} `xml:"body"`
}

type opmlHead struct {
	Title string `xml:"title"`
}

type opmlOutline struct {
	Text     string        `xml:"text,attr"`
	Title    string        `xml:"title,attr"`
	Type     string        `xml:"type,attr"`
	XMLURL   string        `xml:"xmlUrl,attr"`
	Outlines []opmlOutline `xml:"outline"`
}

// Sub is a subscription found in an OPML file.
type Sub struct {
	Title string
	URL   string
}

// ImportOPML reads podcast subscriptions from an OPML file.
func ImportOPML(path string) ([]Sub, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc opmlDoc
	dec := xml.NewDecoder(strings.NewReader(string(raw)))
	dec.Strict = false
	if err := dec.Decode(&doc); err != nil {
		return nil, fmt.Errorf("parse opml: %w", err)
	}
	var out []Sub
	var walk func(os []opmlOutline)
	walk = func(os []opmlOutline) {
		for _, o := range os {
			if o.XMLURL != "" {
				t := o.Title
				if t == "" {
					t = o.Text
				}
				out = append(out, Sub{Title: t, URL: o.XMLURL})
			}
			walk(o.Outlines)
		}
	}
	walk(doc.Body.Outlines)
	return out, nil
}

// ExportOPML writes subscriptions to an OPML file.
func ExportOPML(path string, subs []Sub) error {
	doc := opmlDoc{Version: "2.0"}
	doc.Head.Title = "SopeBox subscriptions"
	for _, s := range subs {
		doc.Body.Outlines = append(doc.Body.Outlines, opmlOutline{Text: s.Title, Title: s.Title, Type: "rss", XMLURL: s.URL})
	}
	out, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append([]byte(xml.Header), out...), 0o644)
}
