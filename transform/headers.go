package transform

import (
	"context"
	"sort"
	"strings"

	"github.com/firfircelik/go-intake"
)

// HeaderStyle is a function that maps a raw header string to a
// normalised one. Several common styles are predefined as variables
// (SnakeCase, CamelCase, KebabCase, LowerCase, UpperCase) and users
// can supply their own.
type HeaderStyle func(string) string

// SnakeCase converts "Product Name" / "product-name" / "ProductName"
// into "product_name".
var SnakeCase HeaderStyle = func(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	// Insert underscores at camelCase boundaries: a lowercase
	// followed by an uppercase. Iterate in reverse so we can insert
	// without invalidating indices.
	b := []byte(s)
	out := make([]byte, 0, len(b)+4)
	prevLower := false
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c == ' ' || c == '-' || c == '.' || c == '/' || c == '\\' || c == '\t' {
			if len(out) > 0 && out[len(out)-1] != '_' {
				out = append(out, '_')
			}
			prevLower = false
			continue
		}
		if c >= 'A' && c <= 'Z' {
			if prevLower && len(out) > 0 && out[len(out)-1] != '_' {
				out = append(out, '_')
			}
			c = c + ('a' - 'A')
			prevLower = false
		} else {
			prevLower = c >= 'a' && c <= 'z'
		}
		out = append(out, c)
	}
	// Collapse repeated underscores.
	final := make([]byte, 0, len(out))
	lastUnderscore := false
	for _, c := range out {
		if c == '_' {
			if !lastUnderscore && len(final) > 0 {
				final = append(final, '_')
			}
			lastUnderscore = true
			continue
		}
		lastUnderscore = false
		final = append(final, c)
	}
	// Trim trailing underscores.
	for len(final) > 0 && final[len(final)-1] == '_' {
		final = final[:len(final)-1]
	}
	return string(final)
}

// CamelCase converts "Product Name" / "product_name" into "productName".
var CamelCase HeaderStyle = func(s string) string {
	parts := splitWords(s)
	if len(parts) == 0 {
		return ""
	}
	var b strings.Builder
	for i, p := range parts {
		if p == "" {
			continue
		}
		if i == 0 {
			b.WriteString(strings.ToLower(p))
			continue
		}
		b.WriteString(strings.ToUpper(p[:1]))
		b.WriteString(strings.ToLower(p[1:]))
	}
	return b.String()
}

// KebabCase converts "Product Name" / "product_name" into "product-name".
var KebabCase HeaderStyle = func(s string) string {
	snake := SnakeCase(s)
	return strings.ReplaceAll(snake, "_", "-")
}

// LowerCase lowercases the input. Spaces and punctuation are
// preserved.
var LowerCase HeaderStyle = strings.ToLower

// UpperCase uppercases the input. Spaces and punctuation are
// preserved.
var UpperCase HeaderStyle = strings.ToUpper

func splitWords(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	var words []string
	var current strings.Builder
	prevLower := false
	prevUpper := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		lower := c >= 'a' && c <= 'z'
		upper := c >= 'A' && c <= 'Z'
		digit := c >= '0' && c <= '9'
		isWord := lower || upper || digit
		if !isWord {
			if current.Len() > 0 {
				words = append(words, current.String())
				current.Reset()
			}
			prevLower = false
			prevUpper = false
			continue
		}
		if upper && prevLower {
			words = append(words, current.String())
			current.Reset()
		} else if lower && prevUpper && current.Len() > 1 {
			last := current.String()
			current.Reset()
			current.WriteString(last[:len(last)-1])
			words = append(words, last[:len(last)-1])
		}
		current.WriteByte(c)
		prevLower = lower
		prevUpper = upper
	}
	if current.Len() > 0 {
		words = append(words, current.String())
	}
	return words
}

// NormalizeHeaders returns a Transformer that rewrites every key of
// each record using the supplied style. On key collisions the
// lexicographically smaller source key wins; later duplicates are
// dropped. Output ordering matches input key ordering.
func NormalizeHeaders(style HeaderStyle) intake.Transformer {
	return headerNormalizer{style: style}
}

type headerNormalizer struct {
	style HeaderStyle
}

func (h headerNormalizer) Apply(_ context.Context, r intake.Record) (intake.Record, error) {
	if r == nil {
		return nil, nil
	}
	keys := make([]string, 0, len(r))
	for k := range r {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make(intake.Record, len(r))
	for _, k := range keys {
		nk := h.style(k)
		if nk == "" {
			continue
		}
		if _, exists := out[nk]; !exists {
			out[nk] = r[k]
		}
	}
	return out, nil
}
