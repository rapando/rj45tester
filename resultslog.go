package main

import (
	"fmt"
	"os"
	"time"
)

// appendResultsLog appends a timestamped, plain-text (no ANSI color) copy
// of the test report to path, creating the file if it doesn't exist. Used
// alongside the colored screen report so results can be compared across
// multiple runs (e.g. testing several cables back to back).
func appendResultsLog(path, device, hardwarePortName string, st ifaceStatus) error {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()

	fmt.Fprintf(f, "===== %s =====\n", time.Now().Format("2006-01-02 15:04:05 MST"))
	if hardwarePortName != "" {
		fmt.Fprintf(f, "Interface: %s (%s)\n", device, hardwarePortName)
	}
	fmt.Fprint(f, buildReport(device, st, false))
	fmt.Fprintln(f)

	return nil
}
