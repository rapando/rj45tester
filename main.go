// rj45tester tests an RJ45 Ethernet cable plugged into a Mac via a
// USB-to-Ethernet adapter. It checks (1) whether all 8 conductors appear
// connected, by seeing whether gigabit (which needs all 4 wire pairs)
// negotiates, and (2) the max link speed negotiated with a router.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "list":
		runList()
	case "test":
		runTest(os.Args[2:])
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "%s\n\n", red(fmt.Sprintf("unknown command: %s", os.Args[1])))
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`rj45tester - RJ45 cable tester via a USB-to-Ethernet adapter

Usage:
  rj45tester list               List wired Ethernet-capable network interfaces
  rj45tester test [-iface enN]  Run the wiring + link-speed test

Flags for 'test':
  -iface string   Interface device to test (e.g. en5). If omitted, the tool
                  auto-detects your USB Ethernet adapter.
  -out string     Results log file to append a plain-text copy of each test
                  to (default "rj45tester-results.txt"). Useful for
                  comparing multiple cables over several runs.
  -no-log         Skip writing to the results log file.

Example:
  1. Plug the USB-to-Ethernet adapter into the Mac.
  2. Plug the RJ45 cable into the adapter and the other end into the router.
  3. Run: rj45tester test`)
}

func runList() {
	ports, err := listHardwarePorts()
	if err != nil {
		fmt.Fprintln(os.Stderr, red("error:"), err)
		os.Exit(1)
	}
	if len(ports) == 0 {
		fmt.Println("No network hardware ports found.")
		return
	}
	fmt.Printf("%-28s %-8s %-20s %s\n", "HARDWARE PORT", "DEVICE", "MAC ADDRESS", "LINK")
	for _, p := range ports {
		link := "-"
		if isEthernetCandidate(p) {
			if st, err := getInterfaceStatus(p.Device); err == nil {
				if st.Active {
					link = green(fmt.Sprintf("active (%s)", st.MediaRaw))
				} else {
					link = red("inactive")
				}
			}
		}
		fmt.Printf("%-28s %-8s %-20s %s\n", p.Name, p.Device, p.MAC, link)
	}
}

func runTest(args []string) {
	fs := flag.NewFlagSet("test", flag.ExitOnError)
	ifaceFlag := fs.String("iface", "", "interface device to test, e.g. en5")
	outFlag := fs.String("out", "rj45tester-results.txt", "results log file to append to")
	noLogFlag := fs.Bool("no-log", false, "skip writing to the results log file")
	if err := fs.Parse(args); err != nil {
		os.Exit(1)
	}

	device := *ifaceFlag
	var autoDetectedName string
	if device == "" {
		d, name, err := autoDetectUSBEthernet()
		if err != nil {
			fmt.Fprintln(os.Stderr, red("error:"), err)
			fmt.Fprintln(os.Stderr, "Run 'rj45tester list' and pass -iface explicitly.")
			os.Exit(1)
		}
		device = d
		autoDetectedName = name
		fmt.Printf("Auto-detected interface: %s (%s)\n\n", device, name)
	}

	status, err := getInterfaceStatus(device)
	if err != nil {
		fmt.Fprintln(os.Stderr, red("error:"), err)
		os.Exit(1)
	}

	fmt.Print(buildReport(device, status, colorEnabled))

	if !*noLogFlag {
		if err := appendResultsLog(*outFlag, device, autoDetectedName, status); err != nil {
			fmt.Fprintln(os.Stderr, red("warning:"), "could not write results log:", err)
			return
		}
		fmt.Printf("\nResults appended to %s\n", *outFlag)
	}
}
