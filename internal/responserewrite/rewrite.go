package responserewrite

import (
	"strings"

	"ds2api/internal/config"
)

type Rule = config.ResponseReplacementRule

func Apply(text string, rules []Rule) string {
	for _, rule := range rules {
		if strings.TrimSpace(rule.From) == "" {
			continue
		}
		text = strings.ReplaceAll(text, rule.From, rule.To)
	}
	return text
}

type StreamReplacer struct {
	rules   []Rule
	pending string
	keep    int
}

func NewStreamReplacer(rules []Rule) *StreamReplacer {
	clean := make([]Rule, 0, len(rules))
	keep := 0
	for _, rule := range rules {
		if strings.TrimSpace(rule.From) == "" {
			continue
		}
		clean = append(clean, rule)
		if n := len(rule.From); n > keep {
			keep = n
		}
	}
	return &StreamReplacer{rules: clean, keep: keep}
}

func (r *StreamReplacer) Push(chunk string) string {
	if r == nil || len(r.rules) == 0 {
		return chunk
	}
	r.pending += chunk
	if len(r.pending) <= r.keep {
		return ""
	}
	emitLen := len(r.pending) - r.keep
	emit := r.pending[:emitLen]
	r.pending = r.pending[emitLen:]
	return Apply(emit, r.rules)
}

func (r *StreamReplacer) Flush() string {
	if r == nil || r.pending == "" {
		return ""
	}
	out := Apply(r.pending, r.rules)
	r.pending = ""
	return out
}
