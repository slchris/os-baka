package sysutil

import (
	"testing"
)

func TestListInterfaces(t *testing.T) {
	ifaces, err := ListInterfaces()
	if err != nil {
		t.Fatalf("ListInterfaces() error = %v", err)
	}
	// Every system should have at least one interface (loopback)
	if len(ifaces) == 0 {
		t.Error("ListInterfaces() returned 0 interfaces, expected at least 1 (loopback)")
	}

	// Check that the loopback interface exists
	foundLoopback := false
	for _, iface := range ifaces {
		if iface.Name == "lo" || iface.Name == "lo0" {
			foundLoopback = true
			// Loopback should have LOOPBACK flag
			hasLoopbackFlag := false
			for _, flag := range iface.Flags {
				if flag == "LOOPBACK" {
					hasLoopbackFlag = true
					break
				}
			}
			if !hasLoopbackFlag {
				t.Error("Loopback interface should have LOOPBACK flag")
			}
			break
		}
	}

	if !foundLoopback {
		t.Log("Warning: Loopback interface (lo/lo0) not found, this is unusual but not fatal on all systems")
	}
}

func TestListInterfacesHaveNames(t *testing.T) {
	ifaces, err := ListInterfaces()
	if err != nil {
		t.Fatalf("ListInterfaces() error = %v", err)
	}

	for _, iface := range ifaces {
		if iface.Name == "" {
			t.Error("ListInterfaces() returned interface with empty name")
		}
	}
}

func TestGetLocalIP(t *testing.T) {
	ip := GetLocalIP()
	// On a machine with network connectivity, this should return a non-empty string
	// In CI without connectivity, it may return empty, so we just check it doesn't panic
	if ip != "" {
		// Validate it looks like an IP
		if len(ip) < 7 { // shortest valid IP: "1.1.1.1"
			t.Errorf("GetLocalIP() = %q, doesn't look like a valid IP", ip)
		}
	} else {
		t.Log("GetLocalIP() returned empty, likely no network connectivity")
	}
}

func TestGetInterfaceIPNonExistent(t *testing.T) {
	ip := GetInterfaceIP("this-interface-does-not-exist-12345")
	if ip != "" {
		t.Errorf("GetInterfaceIP(non-existent) = %q, want empty string", ip)
	}
}

func TestGetInterfaceIPLoopback(t *testing.T) {
	// Try common loopback names
	for _, name := range []string{"lo", "lo0"} {
		ip := GetInterfaceIP(name)
		if ip == "127.0.0.1" {
			// Found it
			return
		}
	}
	t.Log("Could not verify loopback IP, may be named differently on this system")
}

func TestNetworkInterfaceStruct(t *testing.T) {
	ni := NetworkInterface{
		Name:         "eth0",
		HardwareAddr: "aa:bb:cc:dd:ee:ff",
		IPAddresses:  []string{"192.168.1.1/24"},
		Flags:        []string{"UP", "BROADCAST"},
	}

	if ni.Name != "eth0" {
		t.Errorf("Name = %q, want %q", ni.Name, "eth0")
	}
	if ni.HardwareAddr != "aa:bb:cc:dd:ee:ff" {
		t.Errorf("HardwareAddr = %q, want %q", ni.HardwareAddr, "aa:bb:cc:dd:ee:ff")
	}
	if len(ni.IPAddresses) != 1 {
		t.Errorf("IPAddresses len = %d, want 1", len(ni.IPAddresses))
	}
	if len(ni.Flags) != 2 {
		t.Errorf("Flags len = %d, want 2", len(ni.Flags))
	}
}
