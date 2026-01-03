package linepanel

import (
	nt "parcours/entity"
	"parcours/style"
)

var formatters = map[string]nt.Formatter{
	"fourzed": fourzed,
}

// getFormatter returns a formatter for the given format string and width.
// Falls back to Value.String() if format is empty or unknown.
// Wraps result with truncation to width.
func getFormatter(format string, width int) nt.Formatter {
	f := formatters[format]
	if f == nil {
		f = func(v nt.Value) string { return v.String() }
	}
	return func(v nt.Value) string {
		return truncate(f(v), width)
	}
}

const fourzedLayout = "15:04:05.0000"

// fourzed formats a timestamp with 4 decimal places - the sweet spot for logs
func fourzed(val nt.Value) string {
	t, err := val.Time()
	if err != nil {
		return val.String()
	}
	return t.Format(fourzedLayout)
}

func truncate(s string, width int) string {
	if width <= 0 {
		return s
	}
	runes := []rune(s)
	if len(runes) <= width {
		return s
	}
	return string(runes[:width-1]) + style.MutedStyle.Render("…")
}
