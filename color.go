package main

import "os"

const (
	colorReset  = "\033[0m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
)

// colorEnabled is decided once at startup: color is suppressed when NO_COLOR
// is set, TERM=dumb, or stdout/stderr isn't an actual terminal (e.g. piped
// to a file or another program).
var colorEnabled = computeColorEnabled()

func computeColorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" || os.Getenv("TERM") == "dumb" {
		return false
	}
	return isTerminal(os.Stdout) && isTerminal(os.Stderr)
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return (fi.Mode() & os.ModeCharDevice) != 0
}

func colorize(code, s string) string {
	if !colorEnabled {
		return s
	}
	return code + s + colorReset
}

func red(s string) string    { return colorize(colorRed, s) }
func green(s string) string  { return colorize(colorGreen, s) }
func yellow(s string) string { return colorize(colorYellow, s) }

// colorizer returns a coloring function bound to an explicit enabled flag,
// for callers (like report rendering) that need both a colored and a plain
// rendering of the same text rather than relying on the global colorEnabled.
func colorizer(enabled bool) func(code, s string) string {
	return func(code, s string) string {
		if !enabled {
			return s
		}
		return code + s + colorReset
	}
}
