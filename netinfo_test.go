package main

import "testing"

func TestParseIfconfigOutput(t *testing.T) {
	cases := []struct {
		name         string
		out          string
		wantActive   bool
		wantSpeed    int
		wantBaseType string
		wantDuplex   string
	}{
		{
			name: "gigabit full duplex",
			out: `en5: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	ether 00:e0:4c:2d:74:78
	media: autoselect (1000baseT <full-duplex>)
	status: active`,
			wantActive:   true,
			wantSpeed:    1000,
			wantBaseType: "T",
			wantDuplex:   "full-duplex",
		},
		{
			name: "fast ethernet",
			out: `en5: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	media: autoselect (100baseTX <full-duplex>)
	status: active`,
			wantActive:   true,
			wantSpeed:    100,
			wantBaseType: "TX",
			wantDuplex:   "full-duplex",
		},
		{
			name: "no cable plugged in",
			out: `en5: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	media: autoselect (none)
	status: inactive`,
			wantActive: false,
			wantSpeed:  0,
		},
		{
			name: "inactive must not match substring active",
			out: `en3: flags=8863<UP,BROADCAST,SMART,RUNNING,SIMPLEX,MULTICAST> mtu 1500
	media: none
	status: inactive`,
			wantActive: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := parseIfconfigOutput("en5", tc.out)
			if got.Active != tc.wantActive {
				t.Errorf("Active = %v, want %v", got.Active, tc.wantActive)
			}
			if got.SpeedMbps != tc.wantSpeed {
				t.Errorf("SpeedMbps = %d, want %d", got.SpeedMbps, tc.wantSpeed)
			}
			if tc.wantBaseType != "" && got.BaseType != tc.wantBaseType {
				t.Errorf("BaseType = %q, want %q", got.BaseType, tc.wantBaseType)
			}
			if tc.wantDuplex != "" && got.Duplex != tc.wantDuplex {
				t.Errorf("Duplex = %q, want %q", got.Duplex, tc.wantDuplex)
			}
		})
	}
}
