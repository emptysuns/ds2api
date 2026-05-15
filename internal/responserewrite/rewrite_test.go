package responserewrite

import (
	"strings"
	"testing"

	"ds2api/internal/config"
)

func TestApplyLiteralReplacements(t *testing.T) {
	rules := []config.ResponseReplacementRule{
		{From: "<|DEML", To: "<|DSML"},
		{From: "</|DEML", To: "</|DSML"},
	}
	got := Apply("<|DEML|tool_calls></|DEML|tool_calls>", rules)
	want := "<|DSML|tool_calls></|DSML|tool_calls>"
	if got != want {
		t.Fatalf("Apply()=%q want=%q", got, want)
	}
}

func TestApplySkipsEmptyFrom(t *testing.T) {
	rules := []config.ResponseReplacementRule{
		{From: "", To: "x"},
		{From: "   ", To: "y"},
		{From: "a", To: "b"},
	}
	if got := Apply("a", rules); got != "b" {
		t.Fatalf("Apply()=%q want=b", got)
	}
}

func TestApplyNilRulesReturnsInput(t *testing.T) {
	if got := Apply("abc", nil); got != "abc" {
		t.Fatalf("Apply()=%q want=abc", got)
	}
}

func TestStreamReplacerHandlesBoundarySplit(t *testing.T) {
	r := NewStreamReplacer([]config.ResponseReplacementRule{{From: "<|DEML", To: "<|DSML"}})
	p1 := r.Push("prefix <|DE")
	p2 := r.Push("ML|tool_calls>")
	p3 := r.Flush()
	got := p1 + p2 + p3
	want := "prefix <|DSML|tool_calls>"
	if got != want {
		t.Fatalf("stream replacement=%q want=%q parts=[%q,%q,%q]", got, want, p1, p2, p3)
	}
}

func TestStreamReplacerReturnsInputWhenNoRules(t *testing.T) {
	r := NewStreamReplacer(nil)
	if got := r.Push("abc"); got != "abc" {
		t.Fatalf("Push()=%q want=abc", got)
	}
	if got := r.Flush(); got != "" {
		t.Fatalf("Flush()=%q want empty", got)
	}
}

func TestStreamReplacerMultipleRules(t *testing.T) {
	r := NewStreamReplacer([]config.ResponseReplacementRule{
		{From: "<|DEML", To: "<|DSML"},
		{From: "</|DEML", To: "</|DSML"},
	})
	p1 := r.Push("open <|DE")
	p2 := r.Push("ML|tool> close </|DEML|tool_calls>")
	p3 := r.Flush()
	got := p1 + p2 + p3
	want := "open <|DSML|tool> close </|DSML|tool_calls>"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStreamReplacerFlushesPendingTail(t *testing.T) {
	r := NewStreamReplacer([]config.ResponseReplacementRule{{From: "<|DEML", To: "<|DSML"}})
	p1 := r.Push("tail <|DE")
	got := p1 + r.Flush()
	// The pending buffer "<|DE" is a partial match (missing "ML") so no replacement occurs
	want := "tail <|DE"
	if got != want {
		t.Fatalf("Flush() with pending got %q want %q", got, want)
	}
}

func TestStreamReplacerNilReceiverSafe(t *testing.T) {
	var r *StreamReplacer
	if got := r.Push("abc"); got != "abc" {
		t.Fatalf("nil.Push()=%q want=abc", got)
	}
	if got := r.Flush(); got != "" {
		t.Fatalf("nil.Flush()=%q want empty", got)
	}
}

func TestRewrittenDEMLToolCallsParseAsDSML(t *testing.T) {
	rules := []config.ResponseReplacementRule{
		{From: "<|DEML", To: "<|DSML"},
		{From: "</|DEML", To: "</|DSML"},
	}
	text := Apply(`<|DEML|tool_calls><|DEML|invoke name="Bash"><|DEML|parameter name="command"><![CDATA[pwd]]></|DEML|parameter></|DEML|invoke></|DEML|tool_calls>`, rules)
	// Verify the output is valid DSML that the toolcall parser would accept
	if !strings.Contains(text, "<|DSML|tool_calls>") {
		t.Fatalf("expected DSML wrapper, got %q", text)
	}
	if strings.Contains(text, "<|DEML") {
		t.Fatalf("DEML not fully replaced: %q", text)
	}
}
