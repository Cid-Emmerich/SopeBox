package mentions

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/anthropics/anthropic-sdk-go/option"
)

// fakeClaude answers one streamed Messages request with answer and
// records the request it received.
func fakeClaude(t *testing.T, stopReason, answer string) (*httptest.Server, *map[string]any, *http.Header) {
	t.Helper()
	body := map[string]any{}
	hdr := http.Header{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		json.Unmarshal(raw, &body)
		hdr = r.Header.Clone()
		w.Header().Set("Content-Type", "text/event-stream")
		send := func(ev string, v any) {
			b, _ := json.Marshal(v)
			fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev, b)
		}
		send("message_start", map[string]any{"type": "message_start", "message": map[string]any{
			"id": "msg_1", "type": "message", "role": "assistant", "model": "claude-opus-5-5", "content": []any{},
			"usage": map[string]any{"input_tokens": 19433, "output_tokens": 1}}})
		send("content_block_start", map[string]any{"type": "content_block_start", "index": 0, "content_block": map[string]any{"type": "text", "text": ""}})
		for _, part := range []string{answer[:len(answer)/2], answer[len(answer)/2:]} {
			send("content_block_delta", map[string]any{"type": "content_block_delta", "index": 0, "delta": map[string]any{"type": "text_delta", "text": part}})
		}
		send("content_block_stop", map[string]any{"type": "content_block_stop", "index": 0})
		send("message_delta", map[string]any{"type": "message_delta", "delta": map[string]any{"stop_reason": stopReason}, "usage": map[string]any{"output_tokens": 812}})
		send("message_stop", map[string]any{"type": "message_stop"})
	}))
	testOptions = []option.RequestOption{option.WithBaseURL(srv.URL)}
	t.Cleanup(func() { testOptions = nil; srv.Close() })
	return srv, &body, &hdr
}

func TestExtractRequestAndStream(t *testing.T) {
	answer := `{"mentions":[{"t":14,"spoken":"Henry the Eighth","name":"Henry VIII","kind":"person","wiki":"Henry VIII"},{"t":14,"spoken":"Anne Bolin","name":"Anne Boleyn","kind":"person","wiki":"Anne Boleyn"}]}`
	_, body, hdr := fakeClaude(t, "end_turn", answer)
	ep := Episode{Podcast: "The Rest Is History", Title: "The Tudors", Description: "Tom and Dominic on Henry VIII.", Hosts: []string{"Tom Holland", "Dominic Sandbrook"}}
	tl, err := Extract(context.Background(), "sk-test", "", ep, sampleTranscript())
	if err != nil {
		t.Fatal(err)
	}
	if len(tl.Mentions) != 2 || tl.Mentions[0].At != 15.1 || tl.Mentions[1].Name != "Anne Boleyn" {
		t.Fatalf("timeline: %+v", tl.Mentions)
	}
	if tl.Model != "claude-opus-5-5" || tl.InputTokens != 19433 || tl.OutputTokens != 812 || tl.Source != "whisper:base.en" {
		t.Fatalf("metadata: %+v", tl)
	}
	b := *body
	if b["model"] != "claude-opus-5-5" || b["stream"] != true || b["fallbacks"] != "default" {
		t.Fatalf("request: model=%v stream=%v fallbacks=%v", b["model"], b["stream"], b["fallbacks"])
	}
	oc, _ := b["output_config"].(map[string]any)
	format, _ := oc["format"].(map[string]any)
	if oc["effort"] != "medium" || format["type"] != "json_schema" || format["schema"] == nil {
		t.Fatalf("output_config: %v", oc)
	}
	if !strings.Contains(hdr.Get("Anthropic-Beta"), "server-side-fallback-2026-07-01") || hdr.Get("X-Api-Key") != "sk-test" {
		t.Fatalf("headers: beta=%q key=%q", hdr.Get("Anthropic-Beta"), hdr.Get("X-Api-Key"))
	}
	msgs, _ := b["messages"].([]any)
	raw, _ := json.Marshal(msgs)
	for _, want := range []string{"[14] Tom: And of course Henry the Eighth", "Tom Holland, Dominic Sandbrook", "Tom and Dominic on Henry VIII."} {
		if !strings.Contains(string(raw), want) {
			t.Errorf("prompt lacks %q", want)
		}
	}
}

func TestExtractHaikuSkipsEffortAndFallbacks(t *testing.T) {
	_, body, hdr := fakeClaude(t, "end_turn", `{"mentions":[]}`)
	if _, err := Extract(context.Background(), "sk-test", "claude-haiku-4-5", Episode{}, sampleTranscript()); err != nil {
		t.Fatal(err)
	}
	b := *body
	oc, _ := b["output_config"].(map[string]any)
	if _, ok := b["fallbacks"]; ok || oc["effort"] != nil || hdr.Get("Anthropic-Beta") != "" {
		t.Fatalf("haiku request carries unsupported fields: %v beta=%q", b, hdr.Get("Anthropic-Beta"))
	}
}

func TestExtractReportsRefusalAndTruncation(t *testing.T) {
	fakeClaude(t, "refusal", `{"mentions":[]}`)
	if _, err := Extract(context.Background(), "sk-test", "", Episode{}, sampleTranscript()); err == nil || !strings.Contains(err.Error(), "declined") {
		t.Fatalf("refusal: %v", err)
	}
	fakeClaude(t, "max_tokens", `{"mentions":[`+strings.Repeat(" ", 10)+`]}`)
	if _, err := Extract(context.Background(), "sk-test", "", Episode{}, sampleTranscript()); err == nil || !strings.Contains(err.Error(), "cut off") {
		t.Fatalf("truncation: %v", err)
	}
}
