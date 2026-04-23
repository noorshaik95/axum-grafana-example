// Package mentions parses `@username` tokens out of free-form post content.
package mentions

import (
	"regexp"
	"strings"
)

// mentionPattern matches @username — letters, digits, underscore, dot, or hyphen,
// not preceded by a word character (so emails like foo@bar.com do NOT match).
var mentionPattern = regexp.MustCompile(`(^|[^\w])@([A-Za-z0-9][A-Za-z0-9._-]{0,63})`)

// Parse returns the set of unique usernames mentioned in content, preserving
// first-seen order. Case-insensitive dedupe.
func Parse(content string) []string {
	matches := mentionPattern.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(matches))
	out := make([]string, 0, len(matches))
	for _, m := range matches {
		// m[2] is the username capture group
		uname := strings.TrimRight(m[2], ".")
		if uname == "" {
			continue
		}
		key := strings.ToLower(uname)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, uname)
	}
	return out
}
