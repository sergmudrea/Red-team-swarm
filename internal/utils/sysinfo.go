package utils

import (
	"net"
	"os"
	"runtime"
)

// GetHostname returns the system hostname or "unknown" if it cannot be determined.
func GetHostname() string {
	name, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return name
}

// GetOS returns the operating system name (e.g., "linux", "windows").
func GetOS() string {
	return runtime.GOOS
}

// GetInternalIP returns the first non‑loopback IPv4 address of the machine.
// If no suitable address is found, it returns "unknown".
func GetInternalIP() string {
	interfaces, err := net.Interfaces()
	if err != nil {
		return "unknown"
	}
	for _, iface := range interfaces {
		// skip down or loopback
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			ipNet, ok := addr.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipNet.IP
			if ip.To4() != nil && !ip.IsLoopback() {
				return ip.String()
			}
		}
	}
	return "unknown"
}
