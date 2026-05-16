package shared

import (
	"strings"

	"ds2api/internal/responserewrite"
	"ds2api/internal/sse"
)

type StreamAccumulator struct {
	ThinkingEnabled       bool
	SearchEnabled         bool
	StripReferenceMarkers bool
	PreserveToolMarkup    bool

	ResponseReplacer *responserewrite.StreamReplacer

	RawThinking           strings.Builder
	Thinking              strings.Builder
	ToolDetectionThinking strings.Builder
	RawText               strings.Builder
	Text                  strings.Builder
}

type StreamPartDelta struct {
	Type         string
	RawText      string
	VisibleText  string
	CitationOnly bool
}

type StreamAccumulatorResult struct {
	ContentSeen bool
	Parts       []StreamPartDelta
}

func (a *StreamAccumulator) Apply(parsed sse.LineResult) StreamAccumulatorResult {
	out := StreamAccumulatorResult{}
	for _, p := range parsed.ToolDetectionThinkingParts {
		trimmed := sse.TrimContinuationOverlapFromBuilder(&a.ToolDetectionThinking, p.Text)
		if trimmed != "" {
			a.ToolDetectionThinking.WriteString(trimmed)
		}
	}
	for _, p := range parsed.Parts {
		if p.Type == "thinking" {
			delta := a.applyThinkingPart(p.Text)
			if delta.RawText != "" {
				out.ContentSeen = true
			}
			if delta.RawText != "" || delta.VisibleText != "" {
				out.Parts = append(out.Parts, delta)
			}
			continue
		}
		delta := a.applyTextPart(p.Text)
		if delta.RawText != "" {
			out.ContentSeen = true
		}
		if delta.RawText != "" || delta.VisibleText != "" || delta.CitationOnly {
			out.Parts = append(out.Parts, delta)
		}
	}
	return out
}

func (a *StreamAccumulator) applyThinkingPart(text string) StreamPartDelta {
	rawTrimmed := sse.TrimContinuationOverlapFromBuilder(&a.RawThinking, text)
	if rawTrimmed != "" {
		a.RawThinking.WriteString(rawTrimmed)
	}
	delta := StreamPartDelta{Type: "thinking", RawText: rawTrimmed}
	if !a.ThinkingEnabled || rawTrimmed == "" {
		return delta
	}
	cleanedText := CleanVisibleOutputWithPolicy(rawTrimmed, a.StripReferenceMarkers, a.PreserveToolMarkup)
	if cleanedText == "" {
		return delta
	}
	trimmed := sse.TrimContinuationOverlapFromBuilder(&a.Thinking, cleanedText)
	if trimmed == "" {
		return delta
	}
	a.Thinking.WriteString(trimmed)
	delta.VisibleText = trimmed
	return delta
}

func (a *StreamAccumulator) applyTextPart(text string) StreamPartDelta {
	rawTrimmed := sse.TrimContinuationOverlapFromBuilder(&a.RawText, text)
	if rawTrimmed == "" {
		return StreamPartDelta{Type: "text"}
	}
	// Apply response replacements before accumulating raw text.
	rawTrimmed = a.replaceResponseText(rawTrimmed)
	if rawTrimmed == "" {
		return StreamPartDelta{Type: "text"}
	}
	a.RawText.WriteString(rawTrimmed)
	delta := StreamPartDelta{Type: "text", RawText: rawTrimmed}
	if a.SearchEnabled && sse.IsCitation(rawTrimmed) {
		delta.CitationOnly = true
		return delta
	}
	cleanedText := CleanVisibleOutputWithPolicy(rawTrimmed, a.StripReferenceMarkers, a.PreserveToolMarkup)
	trimmed := sse.TrimContinuationOverlapFromBuilder(&a.Text, cleanedText)
	if trimmed == "" {
		return delta
	}
	a.Text.WriteString(trimmed)
	delta.VisibleText = trimmed
	return delta
}

// replaceResponseText applies stream replacements to the given text chunk.
// Returns the replaced text, which may be empty if the chunk was fully consumed
// by the stream replacer's lookahead buffer.
func (a *StreamAccumulator) replaceResponseText(text string) string {
	if a.ResponseReplacer == nil {
		return text
	}
	return a.ResponseReplacer.Push(text)
}

// FlushResponseReplacements flushes any remaining text buffered in the
// stream replacer and appends it to RawText and Text builders. Returns a
// delta for the flushed text (may be empty if nothing was pending).
func (a *StreamAccumulator) FlushResponseReplacements() StreamPartDelta {
	if a.ResponseReplacer == nil {
		return StreamPartDelta{Type: "text"}
	}
	flushed := a.ResponseReplacer.Flush()
	if flushed == "" {
		return StreamPartDelta{Type: "text"}
	}
	a.RawText.WriteString(flushed)
	cleanedText := CleanVisibleOutputWithPolicy(flushed, a.StripReferenceMarkers, a.PreserveToolMarkup)
	trimmed := sse.TrimContinuationOverlapFromBuilder(&a.Text, cleanedText)
	delta := StreamPartDelta{Type: "text", RawText: flushed}
	if trimmed != "" {
		a.Text.WriteString(trimmed)
		delta.VisibleText = trimmed
	}
	if a.SearchEnabled && sse.IsCitation(flushed) {
		delta.CitationOnly = true
	}
	return delta
}
