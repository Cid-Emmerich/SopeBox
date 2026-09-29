package mentions

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/anthropics/anthropic-sdk-go/shared/constant"

	"github.com/Cid-Emmerich/SopeBox/internal/transcript"
)

// DefaultModel is the Claude model used to read transcripts.
const DefaultModel = "claude-opus-5-5"

// testOptions lets tests point the client at a local server.
var testOptions []option.RequestOption

// Episode is what Claude is told about the show besides the transcript.
type Episode struct {
	Podcast     string
	Title       string
	Description string
	Hosts       []string // people declared by the feed, skipped as subjects
}

// HaveCredentials reports whether a Claude API credential is available:
// an explicit key, the standard environment variables, or a profile from
// `ant auth login`.
func HaveCredentials(key string) bool {
	if key != "" || os.Getenv("ANTHROPIC_API_KEY") != "" || os.Getenv("ANTHROPIC_AUTH_TOKEN") != "" || os.Getenv("ANTHROPIC_PROFILE") != "" {
		return true
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	_, err := os.Stat(filepath.Join(dir, "anthropic"))
	return err == nil
}

const systemPrompt = `You prepare a visual companion for podcast episodes. While an episode plays, the listener's screen shows a collage of pictures from Wikipedia: whoever and whatever is being discussed appears the moment it is first said. Your job is to read the timestamped transcript and list those moments.

What to include: specific people, places, events (battles, wars, revolutions, trials), groups (dynasties, armies, parties, companies), works (books, paintings, films, ships, buildings, artefacts) and eras, whenever a picture would help a listener follow along. Include passing references too if a picture would genuinely help; skip ones that are only name-dropped in a list.

What to skip: the hosts and regular guests themselves, advertisements and sponsor reads, housekeeping (subscribe, live tour, membership club), and generic concepts that are not a specific thing ("democracy", "the king" when no particular king is meant).

Resolve every reference to the specific entity meant, using the surrounding conversation and the episode description: "Henry" in an episode about the Tudors is Henry VIII; "the Iron Duke" is Arthur Wellesley, 1st Duke of Wellington. The transcript is machine-generated and often misspells names; give the correct name, not the misspelling.

For each moment:
- t: the number in square brackets at the start of the line where it is said.
- spoken: the words exactly as they appear in that line, misspellings included.
- name: the entity's usual name, short enough for a picture caption.
- kind: person, place, event, group, work, thing or era.
- wiki: the exact title of its English Wikipedia article, as it appears in the article URL but with spaces, when you are confident one exists; otherwise an empty string.
- aliases: every other way the transcript refers to it, copied exactly as transcribed, misspellings included: surnames, first names, titles, nicknames, other spellings ("Teodros", "Theodore", "Napier", "the emperor"). SopeBox searches the transcript for these to notice when the conversation comes back to it, so include a generic phrase like "the emperor" only if, in this episode, it always means this entity. Leave out pronouns.

List each entity once, at its first mention. Order the list by t.`

var schema = map[string]any{
	"type": "object",
	"properties": map[string]any{
		"mentions": map[string]any{
			"type": "array",
			"items": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"t":       map[string]any{"type": "integer"},
					"spoken":  map[string]any{"type": "string"},
					"name":    map[string]any{"type": "string"},
					"kind":    map[string]any{"type": "string", "enum": Kinds},
					"wiki":    map[string]any{"type": "string"},
					"aliases": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
				},
				"required":             []string{"t", "spoken", "name", "kind", "wiki", "aliases"},
				"additionalProperties": false,
			},
		},
	},
	"required":             []string{"mentions"},
	"additionalProperties": false,
}

type rawMention struct {
	T       float64  `json:"t"`
	Spoken  string   `json:"spoken"`
	Name    string   `json:"name"`
	Kind    string   `json:"kind"`
	Wiki    string   `json:"wiki"`
	Aliases []string `json:"aliases"`
}

// FormatTranscript renders the transcript as one line per caption segment,
// each prefixed with its start time in whole seconds: "[754] Tom: …".
func FormatTranscript(tr *transcript.Transcript) string {
	var sb strings.Builder
	for _, s := range tr.Segments {
		text := strings.TrimSpace(s.Text)
		if text == "" {
			continue
		}
		fmt.Fprintf(&sb, "[%d] ", int(s.Start))
		if s.Speaker != "" {
			sb.WriteString(s.Speaker + ": ")
		}
		sb.WriteString(text)
		sb.WriteByte('\n')
	}
	return sb.String()
}

func userPrompt(ep Episode, tr *transcript.Transcript) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Podcast: %s\nEpisode: %s\n", ep.Podcast, ep.Title)
	if len(ep.Hosts) > 0 {
		fmt.Fprintf(&sb, "Hosts and regular people (skip them): %s\n", strings.Join(ep.Hosts, ", "))
	}
	if d := strings.TrimSpace(ep.Description); d != "" {
		if r := []rune(d); len(r) > 3000 {
			d = string(r[:3000]) + "…"
		}
		fmt.Fprintf(&sb, "\n<description>\n%s\n</description>\n", d)
	}
	fmt.Fprintf(&sb, "\n<transcript>\n%s</transcript>\n", FormatTranscript(tr))
	return sb.String()
}

// Extract asks Claude for the episode's timeline. key may be empty, in
// which case the SDK finds credentials itself (environment or profile).
func Extract(ctx context.Context, key, model string, ep Episode, tr *transcript.Transcript) (*Timeline, error) {
	if tr == nil || len(tr.Segments) == 0 {
		return nil, errors.New("no transcript to read")
	}
	if model == "" {
		model = DefaultModel
	}
	opts := append([]option.RequestOption(nil), testOptions...)
	if key != "" {
		opts = append(opts, option.WithAPIKey(key))
	}
	client := anthropic.NewClient(opts...)
	params := anthropic.BetaMessageNewParams{
		Model:     model,
		MaxTokens: 64000,
		System:    []anthropic.BetaTextBlockParam{{Text: systemPrompt}},
		Messages: []anthropic.BetaMessageParam{
			anthropic.NewBetaUserMessage(anthropic.NewBetaTextBlock(userPrompt(ep, tr))),
		},
		OutputConfig: anthropic.BetaOutputConfigParam{
			Format: anthropic.BetaJSONOutputFormatParam{Schema: schema},
		},
	}
	if !strings.HasPrefix(model, "claude-haiku") {
		// Haiku takes neither effort nor server-side fallbacks
		params.OutputConfig.Effort = anthropic.BetaOutputConfigEffortMedium
		// if a safety classifier declines, the API retries on a model it picks
		params.Fallbacks = anthropic.BetaFallbacksParamUnion{OfDefault: constant.ValueOf[constant.Default]()}
		params.Betas = []anthropic.AnthropicBeta{anthropic.AnthropicBetaServerSideFallback2026_07_01}
	}
	stream := client.Beta.Messages.NewStreaming(ctx, params)
	msg := anthropic.BetaMessage{}
	for stream.Next() {
		if err := msg.Accumulate(stream.Current()); err != nil {
			return nil, err
		}
	}
	if err := stream.Err(); err != nil {
		return nil, err
	}
	switch msg.StopReason {
	case anthropic.BetaStopReasonRefusal:
		return nil, errors.New("Claude declined to read this transcript")
	case anthropic.BetaStopReasonMaxTokens:
		return nil, errors.New("Claude's answer was cut off (episode too long?)")
	}
	var text strings.Builder
	for _, b := range msg.Content {
		if t, ok := b.AsAny().(anthropic.BetaTextBlock); ok {
			text.WriteString(t.Text)
		}
	}
	tl, err := parse(text.String(), tr)
	if err != nil {
		return nil, err
	}
	tl.Model = string(msg.Model)
	tl.Source = tr.Source
	tl.Created = time.Now()
	tl.InputTokens = msg.Usage.InputTokens
	tl.OutputTokens = msg.Usage.OutputTokens
	return tl, nil
}

// parse turns Claude's JSON into a timeline with word-accurate times.
func parse(raw string, tr *transcript.Transcript) (*Timeline, error) {
	var out struct {
		Mentions []rawMention `json:"mentions"`
	}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("unreadable answer from Claude: %w", err)
	}
	ms := make([]Mention, 0, len(out.Mentions))
	for _, r := range out.Mentions {
		ms = append(ms, Mention{
			At:      refine(tr, r.T, r.Spoken),
			Name:    strings.TrimSpace(r.Name),
			Kind:    r.Kind,
			Wiki:    strings.TrimSpace(r.Wiki),
			Spoken:  r.Spoken,
			Aliases: r.Aliases,
		})
	}
	tl := &Timeline{Found: ms}
	tl.Rebuild(tr)
	return tl, nil
}
