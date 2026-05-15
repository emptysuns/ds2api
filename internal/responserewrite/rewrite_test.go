package responserewrite

import (
	"testing"

	"ds2api/internal/config"
)

func TestApplyLiteralReplacements(t *testing.T) {
	rules := []config.ResponseReplacementRule{{From: "<|DEML", To: "<|DSML"}, {From: "</|DEML", To: "</|DSML"}}
	got := Apply("<|DEML|tool_calls></|DEML|tool_calls>", rules)
	want := "<|DSML|tool_calls></|DSML|tool_calls>"
	if got != want {
		t.Fatalf("Apply()=%q want=%q", got, want)
	}
}

func TestApplySkipsEmptyFrom(t *testing.T) {
	rules := []config.ResponseReplacementRule{{From: "", To: "x"}, {From: "a", To: "b"}}
	if got := Apply("a", rules); got != "b" {
		t.Fatalf("Apply()=%q want=b", got)
	}
}

func TestApplyPreservesNonEmptyFromWhitespace(t *testing.T) {
	rules := []config.ResponseReplacementRule{{From: " a ", To: "x"}}
	if got := Apply("a a ", rules); got != "ax" {
		t.Fatalf("Apply()=%q want=ax", got)
	}
}

func TestStreamReplacerHandlesBoundarySplit(t *testing.T) {
	r := NewStreamReplacer([]config.ResponseReplacementRule{{From: "<|DEML", To: "<|DSML"}})
	parts := []string{
		r.Push("prefix <|DE"),
		r.Push("ML|tool_calls>"),
		r.Flush(),
	}
	got := parts[0] + parts[1] + parts[2]
	want := "prefix <|DSML|tool_calls>"
	if got != want {
		t.Fatalf("stream replacement=%q want=%q parts=%#v", got, want, parts)
	}
}

func TestStreamReplacerHandlesBoundarySplitAtEnd(t *testing.T) {
	r := NewStreamReplacer([]config.ResponseReplacementRule{{From: "<|DEML", To: "<|DSML"}})
	parts := []string{
		r.Push("<|DE"),
		r.Push("ML"),
		r.Flush(),
	}
	got := parts[0] + parts[1] + parts[2]
	want := "<|DSML"
	if got != want {
		t.Fatalf("stream replacement=%q want=%q parts=%#v", got, want, parts)
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
