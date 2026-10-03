package main

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

type hardwarePort struct {
	Name   string
	Device string
	MAC    string
}

// listHardwarePorts runs `networksetup -listallhardwareports` and parses the
// stanza-based output into a slice of hardwarePort.
func listHardwarePorts() ([]hardwarePort, error) {
	out, err := exec.Command("networksetup", "-listallhardwareports").Output()
	if err != nil {
		return nil, fmt.Errorf("networksetup: %w", err)
	}

	var ports []hardwarePort
	var cur hardwarePort
	flush := func() {
		if cur.Name != "" || cur.Device != "" {
			ports = append(ports, cur)
		}
		cur = hardwarePort{}
	}

	for line := range strings.SplitSeq(string(out), "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
			flush()
		case strings.HasPrefix(line, "Hardware Port:"):
			cur.Name = strings.TrimSpace(strings.TrimPrefix(line, "Hardware Port:"))
		case strings.HasPrefix(line, "Device:"):
			cur.Device = strings.TrimSpace(strings.TrimPrefix(line, "Device:"))
		case strings.HasPrefix(line, "Ethernet Address:"):
			cur.MAC = strings.TrimSpace(strings.TrimPrefix(line, "Ethernet Address:"))
		}
	}
	flush()
	return ports, nil
}

var enDeviceRe = regexp.MustCompile(`^en[0-9]+$`)

// isEthernetCandidate reports whether a hardware port looks like a real,
// physical wired Ethernet interface (as opposed to Wi-Fi, a software
// bridge, or Bluetooth).
func isEthernetCandidate(p hardwarePort) bool {
	if !enDeviceRe.MatchString(p.Device) {
		return false
	}
	lower := strings.ToLower(p.Name)
	if strings.Contains(lower, "wi-fi") || strings.Contains(lower, "wifi") {
		return false
	}
	if strings.Contains(lower, "bridge") {
		return false
	}
	if strings.Contains(lower, "bluetooth") {
		return false
	}
	return true
}

type ifaceStatus struct {
	Device    string
	Active    bool
	MediaRaw  string // full "media: ..." line, trimmed
	SpeedMbps int    // 0 if unknown
	BaseType  string // e.g. "T", "TX", "T4" (the part after "<speed>base")
	Duplex    string // "full-duplex", "half-duplex", or "" if unknown
}

var mediaSpeedRe = regexp.MustCompile(`(\d+)base([A-Za-z0-9\-]+)`)
var mediaDuplexRe = regexp.MustCompile(`<([a-zA-Z\-]+)>`)

// getInterfaceStatus runs `ifconfig <device>` and extracts link status and
// negotiated media (speed/duplex).
func getInterfaceStatus(device string) (ifaceStatus, error) {
	out, err := exec.Command("ifconfig", device).Output()
	if err != nil {
		return ifaceStatus{}, fmt.Errorf("ifconfig %s: %w", device, err)
	}
	return parseIfconfigOutput(device, string(out)), nil
}

// parseIfconfigOutput extracts link status and negotiated media (speed/
// duplex) from the text output of `ifconfig <device>`. Split out from
// getInterfaceStatus so it can be unit-tested without shelling out.
func parseIfconfigOutput(device, out string) ifaceStatus {
	st := ifaceStatus{Device: device}
	for line := range strings.SplitSeq(out, "\n") {
		trimmed := strings.TrimSpace(line)
		switch {
		case strings.HasPrefix(trimmed, "status:"):
			st.Active = strings.TrimSpace(strings.TrimPrefix(trimmed, "status:")) == "active"
		case strings.HasPrefix(trimmed, "media:"):
			st.MediaRaw = trimmed
			if m := mediaSpeedRe.FindStringSubmatch(trimmed); m != nil {
				var mbps int
				fmt.Sscanf(m[1], "%d", &mbps)
				st.SpeedMbps = mbps
				st.BaseType = m[2]
			}
			if m := mediaDuplexRe.FindStringSubmatch(trimmed); m != nil {
				st.Duplex = m[1]
			}
		}
	}
	return st
}

// autoDetectUSBEthernet finds the best wired Ethernet candidate: prefer one
// that currently has an active link, else fall back to the sole candidate.
func autoDetectUSBEthernet() (device string, name string, err error) {
	ports, err := listHardwarePorts()
	if err != nil {
		return "", "", err
	}

	var candidates []hardwarePort
	var usbCandidates []hardwarePort
	for _, p := range ports {
		if !isEthernetCandidate(p) {
			continue
		}
		candidates = append(candidates, p)
		if strings.Contains(strings.ToLower(p.Name), "usb") {
			usbCandidates = append(usbCandidates, p)
		}
	}
	// Prefer adapters explicitly named as USB, since that's what the cable
	// is plugged into; only fall back to other wired interfaces (e.g.
	// Thunderbolt Ethernet) if no USB-named port exists.
	if len(usbCandidates) > 0 {
		candidates = usbCandidates
	}
	if len(candidates) == 0 {
		return "", "", fmt.Errorf("no wired Ethernet adapter found; plug in your USB-to-Ethernet adapter")
	}

	for _, c := range candidates {
		st, err := getInterfaceStatus(c.Device)
		if err == nil && st.Active {
			return c.Device, c.Name, nil
		}
	}

	if len(candidates) == 1 {
		return candidates[0].Device, candidates[0].Name, nil
	}

	var names []string
	for _, c := range candidates {
		names = append(names, fmt.Sprintf("%s (%s)", c.Device, c.Name))
	}
	return "", "", fmt.Errorf("multiple Ethernet adapters found with no active link: %s", strings.Join(names, ", "))
}
