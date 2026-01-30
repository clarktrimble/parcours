package detail

import "strings"

/*
  1. Long string values
  {"body": "...10000 chars of data..."}
  If limit falls inside that long string, we won't find a safe point and fall back to dumb truncation. The code currently only
  considers commas and closing brackets as safe points - not mid-string.

  2. Performance
  We scan character by character and copy stack/path at each safe point. For 10KB, that's fine. But it's O(n) with some allocations
  per safe point.

  3. No safe point before limit
  If the first safe point is at byte 2000 but limit is 999, we'd scan to 2000+ looking for a safe point. The "soft limit" could
  overshoot significantly.

  Potential improvements:

  // 1. Add mid-string truncation as fallback
  if i >= softLimit*2 && lastSafePos == 0 {
      // We've gone too far, truncate the current string if in one
      if js.inString {
          // Close string here, mark as truncated
      }
  }

  // 2. Track string start position
  // So we can truncate a long string cleanly

  // 3. Set a hard limit (e.g., 2x soft limit)
  // Beyond which we just fall back to dumb truncation

  For your typical limits (999-9999):
  - Should work well for structured logs with many small-ish fields
  - Watch out for fields with huge embedded JSON (like your body field)
  - The body field is exactly the case where dumb truncation happens today
*/

// SmartTruncate truncates JSON at a clean boundary near the soft limit.
// Returns valid JSON with truncation marker. Falls back to simple
// truncation if the input isn't valid JSON structure.
func SmartTruncate(s string, softLimit int) string {
	if len(s) <= softLimit {
		return s
	}

	result, ok := trySmartTruncate(s, softLimit)
	if ok {
		return result
	}

	// Fall back to simple truncation
	return s[:softLimit] + "--truncated--"
}

func trySmartTruncate(s string, softLimit int) (string, bool) {
	js := &jsonScanner{}

	// Track the last safe truncation point
	lastSafePos := 0
	lastSafeStack := []rune{}
	lastSafePath := []string{}
	lastSafeKey := ""

	for i, r := range s {
		js.processRune(r)

		// Check for safe truncation points (after complete values)
		if isSafeTruncationPoint(js, r) {
			lastSafePos = i + 1
			lastSafeStack = copyRuneSlice(js.stack)
			lastSafePath = copyStringSlice(js.path)
			lastSafeKey = js.currentKey
		}

		// Once we're past the soft limit and have a safe point, use it
		if i >= softLimit && lastSafePos > 0 {
			break
		}
	}

	// No safe point found
	if lastSafePos == 0 {
		return "", false
	}

	// Build the truncated result
	var result strings.Builder
	result.WriteString(s[:lastSafePos])

	// Add truncation marker as a field if in object, or as value if in array
	if len(lastSafeStack) > 0 {
		top := lastSafeStack[len(lastSafeStack)-1]
		if top == '{' {
			// Need comma if last char wasn't already a comma
			lastChar := s[lastSafePos-1]
			if lastChar != ',' && lastChar != '{' {
				result.WriteRune(',')
			}
			result.WriteString(`"_truncated":"`)
			result.WriteString(buildPath(lastSafePath, lastSafeKey))
			result.WriteString(`"`)
		} else {
			// In array - add as value
			lastChar := s[lastSafePos-1]
			if lastChar != ',' && lastChar != '[' {
				result.WriteRune(',')
			}
			result.WriteString(`"--truncated--"`)
		}
	}

	// Close all open structures
	for i := len(lastSafeStack) - 1; i >= 0; i-- {
		if lastSafeStack[i] == '{' {
			result.WriteRune('}')
		} else {
			result.WriteRune(']')
		}
	}

	return result.String(), true
}

// isSafeTruncationPoint returns true if this is a good place to truncate.
func isSafeTruncationPoint(js *jsonScanner, r rune) bool {
	// Not safe if inside a string
	if js.inString {
		return false
	}

	// Safe after commas (between values)
	if r == ',' {
		return true
	}

	// Safe after closing brackets (complete substructure)
	if r == '}' || r == ']' {
		return true
	}

	return false
}

func buildPath(path []string, currentKey string) string {
	if currentKey == "" {
		return strings.Join(path, ".")
	}
	if len(path) == 0 {
		return currentKey
	}
	return strings.Join(path, ".") + "." + currentKey
}

func copyRuneSlice(s []rune) []rune {
	c := make([]rune, len(s))
	copy(c, s)
	return c
}

func copyStringSlice(s []string) []string {
	c := make([]string, len(s))
	copy(c, s)
	return c
}
