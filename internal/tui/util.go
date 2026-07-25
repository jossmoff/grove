package tui

import (
	"fmt"
	"time"
)

// ago renders an epoch-seconds commit time as a coarse relative string.
func ago(unix int64) string {
	if unix == 0 {
		return ""
	}
	d := time.Since(time.Unix(unix, 0))
	switch {
	case d < time.Hour:
		return "just now"
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	default:
		return fmt.Sprintf("%dmo ago", int(d.Hours()/24/30))
	}
}
