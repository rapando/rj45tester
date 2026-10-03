package main

import (
	"fmt"
	"strings"
)

// buildReport renders the two-part test report: (1) wiring integrity,
// inferred from whether gigabit (which requires all 4 pairs / 8 wires)
// negotiated, and (2) the max link speed seen against the router.
//
// useColor controls whether ANSI color codes are embedded. Callers render
// twice: once with useColor matching the terminal for screen output, and
// once with useColor=false for the plain-text results log file.
func buildReport(device string, st ifaceStatus, useColor bool) string {
	var b strings.Builder

	fmt.Fprintf(&b, "Interface: %s\n", device)
	if st.MediaRaw != "" {
		fmt.Fprintf(&b, "Raw media: %s\n", st.MediaRaw)
	}
	b.WriteString("\n")

	b.WriteString("1) Wiring integrity (8 conductors)\n")
	b.WriteString("-----------------------------------\n")
	b.WriteString(wiringDiagnosis(st, useColor))

	b.WriteString("\n")
	b.WriteString("2) Link speed vs. router\n")
	b.WriteString("-------------------------\n")
	b.WriteString(speedDiagnosis(st, useColor))

	b.WriteString("\n")
	b.WriteString("3) Per-pair breakdown (T-568B)\n")
	b.WriteString("-------------------------------\n")
	b.WriteString(connectionsTable(testAllConnections(st), useColor))

	return b.String()
}

// connectionsTable renders the per-pair results from testAllConnections as
// a readable table, with the status column colorized per useColor.
func connectionsTable(pairs []pairResult, useColor bool) string {
	c := colorizer(useColor)
	statusColor := map[pairStatus]string{
		statusOK:      colorGreen,
		statusSuspect: colorYellow,
		statusFail:    colorRed,
		statusUnknown: colorYellow,
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%-5s %-6s %-24s %-9s %s\n", "PIN", "PAIR", "T-568B COLOR", "STATUS", "NOTE")
	for _, p := range pairs {
		// Pad the plain text to the column width first, then colorize -
		// coloring after padding would let the invisible ANSI bytes count
		// toward %-9s's width and throw off the NOTE column's alignment.
		paddedStatus := fmt.Sprintf("%-9s", string(p.Status))
		status := c(statusColor[p.Status], paddedStatus)
		fmt.Fprintf(&b, "%-5s %-6d %-24s %s %s\n", p.Pins, p.Pair, p.Colors, status, p.Note)
	}
	return b.String()
}

func wiringDiagnosis(st ifaceStatus, useColor bool) string {
	c := colorizer(useColor)
	var b strings.Builder
	line := func(code, text string) {
		b.WriteString(c(code, text))
		b.WriteString("\n")
	}

	if !st.Active {
		line(colorRed, "RESULT: FAIL - no link detected.")
		b.WriteString("The adapter reports status: inactive. Either the cable is not fully\n")
		b.WriteString("seated at one end, or pins 1,2,3,6 (the pairs needed for link pulses\n")
		b.WriteString("on any speed) are broken/open. Check both RJ45 ends and the router port.\n")
		return b.String()
	}

	switch {
	case st.SpeedMbps >= 1000 && st.Duplex == "full-duplex":
		line(colorGreen, "RESULT: PASS - all 8 wires appear connected.")
		b.WriteString("Gigabit (1000BASE-T) full-duplex requires all 4 pairs (8 wires) to be\n")
		b.WriteString("intact, so a clean 1000baseT full-duplex link is strong evidence the\n")
		b.WriteString("full 8-conductor path is good.\n")

	case st.SpeedMbps >= 1000:
		line(colorYellow, fmt.Sprintf("RESULT: INCONCLUSIVE - negotiated %dBASE-%s but duplex is %q.", st.SpeedMbps, st.BaseType, st.Duplex))
		b.WriteString("Gigabit normally only runs full-duplex; an unexpected duplex reading\n")
		b.WriteString("suggests a flaky pair rather than a clean fail. Re-seat both ends and re-test.\n")

	case st.SpeedMbps == 100 || st.SpeedMbps == 10:
		line(colorYellow, fmt.Sprintf("RESULT: SUSPECT - only %dBASE-%s negotiated, not gigabit.", st.SpeedMbps, st.BaseType))
		b.WriteString("10/100BASE-T only use pins 1,2,3,6 (2 of the 4 pairs). Since gigabit\n")
		b.WriteString("didn't negotiate, pins 4,5,7,8 (the other 2 pairs) are the most likely\n")
		b.WriteString("suspects for an open/broken wire or bad crimp - but this could also\n")
		b.WriteString("mean the router port itself only supports 10/100. To isolate the cause:\n")
		b.WriteString("  - try this same cable on a known-gigabit-capable port\n")
		b.WriteString("  - try a known-good cable on this same router port\n")

	default:
		line(colorYellow, fmt.Sprintf("RESULT: UNKNOWN - could not parse negotiated speed from media string %q.", st.MediaRaw))
	}

	return b.String()
}

func speedDiagnosis(st ifaceStatus, useColor bool) string {
	c := colorizer(useColor)
	var b strings.Builder
	line := func(code, text string) {
		b.WriteString(c(code, text))
		b.WriteString("\n")
	}

	if !st.Active {
		b.WriteString("No link to the router, so no speed was negotiated.\n")
		return b.String()
	}
	if st.SpeedMbps == 0 {
		b.WriteString("Link is active but speed could not be determined from ifconfig output.\n")
		return b.String()
	}

	fmt.Fprintf(&b, "Negotiated max speed: %d Mbps (%dBASE-%s, %s)\n", st.SpeedMbps, st.SpeedMbps, st.BaseType, orUnknown(st.Duplex))

	switch {
	case st.SpeedMbps >= 1000:
		line(colorGreen, "That's Gigabit Ethernet - the best this cable/port combination supports.")
	case st.SpeedMbps == 100:
		line(colorYellow, "Fast Ethernet (100 Mbps) - below gigabit. Could be cable damage, a")
		line(colorYellow, "short/open pair, excessive cable length, or the router port's own limit.")
	case st.SpeedMbps == 10:
		line(colorRed, "10 Mbps - well below modern expectations. Strongly suggests a cable or")
		line(colorRed, "connector problem (or a very old/limited router port).")
	default:
		line(colorYellow, fmt.Sprintf("Unusual negotiated speed (%d Mbps) - double check the router port.", st.SpeedMbps))
	}

	b.WriteString("\n")
	b.WriteString("Note: since the router is offline, this is the NIC's auto-negotiated link\n")
	b.WriteString("rate (the real ceiling for that link), not a measured application-layer\n")
	b.WriteString("throughput test.\n")

	return b.String()
}

func orUnknown(s string) string {
	if s == "" {
		return "duplex unknown"
	}
	return s
}
