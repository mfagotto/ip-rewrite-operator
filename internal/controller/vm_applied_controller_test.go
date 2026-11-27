/*
Copyright 2026 The ip-rewrite-operator Authors.
Licensed under the Apache License, Version 2.0.
*/

package controller

import (
	"testing"

	virtv1 "kubevirt.io/api/core/v1"
)

func TestParseAnnotationIPv4(t *testing.T) {
	cases := map[string]string{
		"192.168.100.60/24;192.168.100.1": "192.168.100.60",
		"10.0.2.100/24":                   "10.0.2.100",
		"10.0.2.100":                      "10.0.2.100",
		"  192.168.1.1/32;gw  ":           "192.168.1.1",
		"not-an-ip":                       "",
		"fe80::1/64":                      "",
		"":                                "",
	}
	for in, want := range cases {
		if got := parseAnnotationIPv4(in); got != want {
			t.Errorf("parseAnnotationIPv4(%q)=%q want %q", in, got, want)
		}
	}
}

func TestGuestReportsExpectedIPv4(t *testing.T) {
	prefer := "02:00:00:aa:bb:60"
	ifaces := []virtv1.VirtualMachineInstanceNetworkInterface{
		{
			InterfaceName: "lo",
			IP:            "127.0.0.1",
			IPs:           []string{"127.0.0.1"},
			InfoSource:    "guest-agent",
		},
		{
			InterfaceName: "Ethernet",
			MAC:           prefer,
			IP:            "192.168.100.60",
			IPs:           []string{"192.168.100.60", "fe80::1"},
			InfoSource:    "guest-agent",
		},
	}

	if !GuestReportsExpectedIPv4(ifaces, []string{"192.168.100.60"}, prefer) {
		t.Fatal("expected MAC-matched Ethernet IP to stamp")
	}
	if GuestReportsExpectedIPv4(ifaces, []string{"192.168.100.99"}, prefer) {
		t.Fatal("mismatched IP must not stamp")
	}

	// Loopback-only must not stamp even if IP somehow matches desired.
	loOnly := []virtv1.VirtualMachineInstanceNetworkInterface{{
		InterfaceName: "Loopback Pseudo-Interface",
		IP:            "192.168.100.60",
		IPs:           []string{"192.168.100.60"},
		InfoSource:    "guest-agent",
	}}
	if GuestReportsExpectedIPv4(loOnly, []string{"192.168.100.60"}, "") {
		t.Fatal("loopback iface must not stamp")
	}

	// Prefer eth0/Ethernet by name when MAC unknown.
	byName := []virtv1.VirtualMachineInstanceNetworkInterface{
		{InterfaceName: "eth1", IP: "10.0.0.1", IPs: []string{"10.0.0.1"}, InfoSource: "guest-agent"},
		{InterfaceName: "eth0", IP: "192.168.100.65", IPs: []string{"192.168.100.65"}, InfoSource: "guest-agent"},
	}
	if !GuestReportsExpectedIPv4(byName, []string{"192.168.100.65"}, "") {
		t.Fatal("eth0 primary should match")
	}
	// Wrong IP on eth0 should fail even if eth1 has something else.
	if GuestReportsExpectedIPv4(byName, []string{"10.0.0.1"}, "") {
		// eth0 is preferred; 10.0.0.1 is only on eth1 → should fail
		t.Fatal("should prefer eth0 and reject eth1-only IP")
	}
}

func TestDesiredIPv4s(t *testing.T) {
	anns := map[string]string{
		"iprewrite.reip.io/eth0-ip":  "192.168.100.60/24;192.168.100.1",
		"iprewrite.reip.io/dns":      "8.8.8.8",
		"iprewrite.reip.io/applied":  "deadbeef",
		"unrelated":                  "x",
	}
	got := desiredIPv4s(anns)
	if len(got) != 1 || got[0] != "192.168.100.60" {
		t.Fatalf("desiredIPv4s=%v", got)
	}
}
