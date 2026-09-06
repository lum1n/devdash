package tui

import (
	"regexp"
	"strings"
)

var headingRe = regexp.MustCompile(`^#{1,6}\s+`)

func mdPrefixLine(content string, line int, prefix string, replaceHeading bool) string {
	lines := strings.Split(content, "\n")
	if line < 0 || line >= len(lines) {
		return content
	}
	body := lines[line]
	if replaceHeading {
		body = headingRe.ReplaceAllString(body, "")
	}
	if !strings.HasPrefix(body, prefix) {
		body = prefix + body
	}
	lines[line] = body
	return strings.Join(lines, "\n")
}

func mdWrap(content, selected, before, after, fallback string) string {
	if selected != "" {
		return strings.Replace(content, selected, before+selected+after, 1)
	}
	if fallback == "" {
		fallback = "text"
	}
	return content + before + fallback + after
}
