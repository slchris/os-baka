package sysutil

import (
	"net"
)

type NetworkInterface struct {
	Name         string   `json:"name"`
	HardwareAddr string   `json:"mac_address"`
	IPAddresses  []string `json:"ip_addresses"`
	Flags        []string `json:"flags"`
}

// ListInterfaces returns a list of network interfaces on the system
func ListInterfaces() ([]NetworkInterface, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}

	var results []NetworkInterface

	for _, i := range ifaces {
		// Skip down interfaces or loopback if desired, but user might want to debug
		// For now we list everything, but we can filter active ones in frontend or here

		var flags []string
		if i.Flags&net.FlagUp != 0 {
			flags = append(flags, "UP")
		}
		if i.Flags&net.FlagBroadcast != 0 {
			flags = append(flags, "BROADCAST")
		}
		if i.Flags&net.FlagLoopback != 0 {
			flags = append(flags, "LOOPBACK")
		}
		if i.Flags&net.FlagMulticast != 0 {
			flags = append(flags, "MULTICAST")
		}

		addrs, err := i.Addrs()
		var ips []string
		if err == nil {
			for _, addr := range addrs {
				// We usually only care about the IP, not the mask in this simple view
				// addr.String() returns "192.168.1.5/24"
				ips = append(ips, addr.String())
			}
		}

		results = append(results, NetworkInterface{
			Name:         i.Name,
			HardwareAddr: i.HardwareAddr.String(),
			IPAddresses:  ips,
			Flags:        flags,
		})
	}

	return results, nil
}

// GetLocalIP returns the preferred outbound ip of this machine
func GetLocalIP() string {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return ""
	}
	defer func() { _ = conn.Close() }()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String()
}

// GetInterfaceIP returns the first IPv4 address of a named interface
func GetInterfaceIP(name string) string {
	iface, err := net.InterfaceByName(name)
	if err != nil {
		return ""
	}
	addrs, err := iface.Addrs()
	if err != nil {
		return ""
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return ""
}
