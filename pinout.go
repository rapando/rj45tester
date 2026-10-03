package main

import "fmt"

// pairStatus is the per-pair verdict testAllConnections assigns.
type pairStatus string

const (
	statusOK      pairStatus = "OK"
	statusSuspect pairStatus = "SUSPECT"
	statusFail    pairStatus = "FAIL"
	statusUnknown pairStatus = "UNKNOWN"
)

// pairResult describes one T-568B wire pair and what the negotiated link
// implies about its health.
type pairResult struct {
	Pair   int    // T-568B pair number (1-4)
	Pins   string // RJ45 pin numbers this pair occupies
	Colors string // T-568B wire colors for this pair
	Role   string // what this pair is used for at 10/100 vs 1000BASE-T
	Status pairStatus
	Note   string
}

// t568bPairs is the fixed T-568B pinout: which RJ45 pins and wire colors
// make up each of the 4 twisted pairs, and what they're used for.
func t568bPairs() []pairResult {
	return []pairResult{
		{Pair: 2, Pins: "1,2", Colors: "White/Orange, Orange", Role: "TX pair (10/100); data pair (1000)"},
		{Pair: 3, Pins: "3,6", Colors: "White/Green, Green", Role: "RX pair (10/100); data pair (1000)"},
		{Pair: 1, Pins: "4,5", Colors: "Blue, White/Blue", Role: "unused at 10/100; data pair (1000 only)"},
		{Pair: 4, Pins: "7,8", Colors: "White/Brown, Brown", Role: "unused at 10/100; data pair (1000 only)"},
	}
}

// testAllConnections checks all 8 conductors by mapping the NIC's
// negotiated link (status + speed + duplex) onto the 4 T-568B pairs it
// implies are working, since there's no per-pin continuity hardware here.
//
//   - No link at all: pairs 2/3 (pins 1,2,3,6) are required for ANY link
//     speed, so they're the likely fault; pairs 1/4 can't be assessed
//     until a basic link exists.
//   - 10/100BASE-T: pairs 2/3 are confirmed good (they carried the link);
//     pairs 1/4 are untested by this speed, so remain suspect since
//     gigabit - which needs them - didn't negotiate.
//   - 1000BASE-T full-duplex: all 4 pairs are confirmed, since gigabit
//     cannot negotiate without all 8 conductors intact.
func testAllConnections(st ifaceStatus) []pairResult {
	pairs := t568bPairs()

	for i := range pairs {
		isCorePair := pairs[i].Pair == 2 || pairs[i].Pair == 3

		switch {
		case !st.Active:
			if isCorePair {
				pairs[i].Status = statusFail
				pairs[i].Note = "Required for link at any speed - most likely cause of the no-link result."
			} else {
				pairs[i].Status = statusUnknown
				pairs[i].Note = "Can't be assessed until pairs 2/3 establish a basic link first."
			}

		case st.SpeedMbps >= 1000 && st.Duplex == "full-duplex":
			pairs[i].Status = statusOK
			pairs[i].Note = "Confirmed - 1000BASE-T cannot negotiate without this pair intact."

		case st.SpeedMbps >= 1000:
			pairs[i].Status = statusSuspect
			pairs[i].Note = fmt.Sprintf("Gigabit negotiated but duplex is %q, not full - treat as unconfirmed.", st.Duplex)

		case st.SpeedMbps == 100 || st.SpeedMbps == 10:
			if isCorePair {
				pairs[i].Status = statusOK
				pairs[i].Note = fmt.Sprintf("Confirmed - carried the %dBASE-T link.", st.SpeedMbps)
			} else {
				pairs[i].Status = statusSuspect
				pairs[i].Note = "Not used at this speed, and gigabit (which needs it) didn't negotiate."
			}

		default:
			pairs[i].Status = statusUnknown
			pairs[i].Note = "Could not determine from the negotiated media string."
		}
	}

	return pairs
}
