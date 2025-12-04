// Package llmjson provides robust JSON extraction from LLM responses.
package llmjson

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// ParseMethod indicates how the JSON was extracted.
type ParseMethod string

const (
	// ParseMethodDirect means JSON was parsed directly.
	ParseMethodDirect ParseMethod = "direct"
	// ParseMethodExtracted means JSON was extracted from surrounding text.
	ParseMethodExtracted ParseMethod = "extracted"
	// ParseMethodLenient means JSON required error recovery.
	ParseMethodLenient ParseMethod = "lenient"
	// ParseMethodFailed means JSON extraction failed.
	ParseMethodFailed ParseMethod = "failed"
)

// Result contains the parsing result and metadata.
type Result[T any] struct {
	Value   T
	Method  ParseMethod
	Warning string
	Raw     string // Original raw string
}

// ExtractJSON attempts to extract and parse JSON from an LLM response.
// It tries multiple strategies in order:
// 1. Direct unmarshal
// 2. Extract JSON from markdown code blocks
// 3. Find first `{` to matching `}`
// 4. Lenient parsing with recovery
func ExtractJSON[T any](raw string) (*Result[T], error) {
	var out T
	result := &Result[T]{
		Raw: raw,
	}

	// Strategy 1: Direct unmarshal
	if err := json.Unmarshal([]byte(raw), &out); err == nil {
		result.Value = out
		result.Method = ParseMethodDirect
		return result, nil
	}

	// Strategy 2: Extract from markdown code blocks
	snippet := extractFromCodeBlock(raw)
	if snippet != "" {
		if err := json.Unmarshal([]byte(snippet), &out); err == nil {
			result.Value = out
			result.Method = ParseMethodExtracted
			result.Warning = "JSON was extracted from markdown code block"
			return result, nil
		}
	}

	// Strategy 3: Find JSON segment (first { to matching })
	snippet = findJSONSegment(raw)
	if snippet != "" {
		if err := json.Unmarshal([]byte(snippet), &out); err == nil {
			result.Value = out
			result.Method = ParseMethodExtracted
			result.Warning = "JSON was extracted from surrounding text"
			return result, nil
		}
	}

	// Strategy 4: Lenient parsing - try to fix common issues
	fixed := attemptJSONRecovery(raw)
	if fixed != "" {
		if err := json.Unmarshal([]byte(fixed), &out); err == nil {
			result.Value = out
			result.Method = ParseMethodLenient
			result.Warning = "JSON required error recovery"
			return result, nil
		}
	}

	// All strategies failed
	result.Method = ParseMethodFailed
	return nil, fmt.Errorf("failed to extract JSON from LLM response: no valid JSON found")
}

// MustExtractJSON extracts JSON or panics. For use in tests only.
func MustExtractJSON[T any](raw string) T {
	result, err := ExtractJSON[T](raw)
	if err != nil {
		panic(fmt.Sprintf("MustExtractJSON failed: %v", err))
	}
	return result.Value
}

// extractFromCodeBlock extracts JSON from markdown code blocks.
func extractFromCodeBlock(raw string) string {
	// Try ```json ... ```
	jsonBlockRe := regexp.MustCompile("(?s)```(?:json)?\\s*\\n?(.*?)```")
	matches := jsonBlockRe.FindStringSubmatch(raw)
	if len(matches) >= 2 {
		return strings.TrimSpace(matches[1])
	}
	return ""
}

// findJSONSegment finds the first complete JSON object in the string.
func findJSONSegment(raw string) string {
	// Find first {
	start := strings.Index(raw, "{")
	if start == -1 {
		// Try array
		start = strings.Index(raw, "[")
		if start == -1 {
			return ""
		}
	}

	// Determine if we're looking for object or array
	isArray := raw[start] == '['
	openChar, closeChar := '{', '}'
	if isArray {
		openChar, closeChar = '[', ']'
	}

	// Find matching closing bracket
	depth := 0
	inString := false
	escaped := false

	for i := start; i < len(raw); i++ {
		c := raw[i]

		if escaped {
			escaped = false
			continue
		}

		if c == '\\' {
			escaped = true
			continue
		}

		if c == '"' {
			inString = !inString
			continue
		}

		if inString {
			continue
		}

		if c == byte(openChar) {
			depth++
		} else if c == byte(closeChar) {
			depth--
			if depth == 0 {
				return raw[start : i+1]
			}
		}
	}

	return ""
}

// attemptJSONRecovery tries to fix common JSON issues.
func attemptJSONRecovery(raw string) string {
	// First extract what looks like JSON
	snippet := findJSONSegment(raw)
	if snippet == "" {
		snippet = raw
	}

	// Fix 1: Trailing commas
	trailingCommaRe := regexp.MustCompile(`,\s*([}\]])`)
	snippet = trailingCommaRe.ReplaceAllString(snippet, "$1")

	// Fix 2: Single quotes to double quotes (outside of already-quoted strings)
	// This is tricky, so we only do it if there are no double-quoted strings
	if !strings.Contains(snippet, `"`) && strings.Contains(snippet, `'`) {
		snippet = strings.ReplaceAll(snippet, `'`, `"`)
	}

	// Fix 3: Unquoted keys (common in some LLM outputs)
	// Match word: followed by value
	unquotedKeyRe := regexp.MustCompile(`(?m)^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*:`)
	snippet = unquotedKeyRe.ReplaceAllString(snippet, `"$1":`)

	// Fix 4: JavaScript-style comments
	lineCommentRe := regexp.MustCompile(`(?m)//.*$`)
	snippet = lineCommentRe.ReplaceAllString(snippet, "")
	blockCommentRe := regexp.MustCompile(`/\*.*?\*/`)
	snippet = blockCommentRe.ReplaceAllString(snippet, "")

	// Trim whitespace
	snippet = strings.TrimSpace(snippet)

	return snippet
}

// IsValidJSON checks if a string is valid JSON.
func IsValidJSON(s string) bool {
	var v any
	return json.Unmarshal([]byte(s), &v) == nil
}

// PrettyPrint formats JSON with indentation.
func PrettyPrint(v any) (string, error) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// ExtractField extracts a specific field from JSON without full parsing.
// Useful for quick extraction of known fields.
func ExtractField(raw, field string) (string, bool) {
	// Use regex for quick extraction
	pattern := fmt.Sprintf(`"%s"\s*:\s*"([^"]*)"`, regexp.QuoteMeta(field))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(raw)
	if len(matches) >= 2 {
		return matches[1], true
	}
	return "", false
}

// ExtractIntField extracts an integer field from JSON.
func ExtractIntField(raw, field string) (int, bool) {
	pattern := fmt.Sprintf(`"%s"\s*:\s*(-?\d+)`, regexp.QuoteMeta(field))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(raw)
	if len(matches) >= 2 {
		var val int
		if _, err := fmt.Sscanf(matches[1], "%d", &val); err == nil {
			return val, true
		}
	}
	return 0, false
}

// ExtractFloatField extracts a float field from JSON.
func ExtractFloatField(raw, field string) (float64, bool) {
	pattern := fmt.Sprintf(`"%s"\s*:\s*(-?\d+\.?\d*)`, regexp.QuoteMeta(field))
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(raw)
	if len(matches) >= 2 {
		var val float64
		if _, err := fmt.Sscanf(matches[1], "%f", &val); err == nil {
			return val, true
		}
	}
	return 0, false
}
