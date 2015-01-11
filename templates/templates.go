package templates

import (
	"time"
)

func formatTime(t time.Time) string {
	const layout = "2006-01-02 15:04:05 -0700"
	if t.IsZero() {
		return ""
	} else {
		return t.Format(layout)
	}
}
