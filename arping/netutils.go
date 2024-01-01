package arping

import (
	"fmt"
	"net"
)

func findIPInNetworkFromIface(dstIP net.IP, iface net.Interface) (net.IP, error) {
	addrs, err := iface.Addrs()

	if err != nil {
		return nil, err
	}

	for _, a := range addrs {
		if ipnet, ok := a.(*net.IPNet); ok {
			if ipnet.Contains(dstIP) {
				return ipnet.IP, nil
			}
		}
	}
	return nil, fmt.Errorf("iface: '%s' can't reach ip: '%s'", iface.Name, dstIP)
}

func findUsableInterfaceForNetwork(dstIP net.IP) (*net.Interface, error) {
	ifaces, err := net.Interfaces()

	if err != nil {
		return nil, err
	}

	isDown := func(iface net.Interface) bool {
		return iface.Flags&1 == 0
	}

	hasAddressInNetwork := func(iface net.Interface) bool {
		if _, err := findIPInNetworkFromIface(dstIP, iface); err != nil {
			return false
		}
		return true
	}

	for _, iface := range ifaces {
		if isDown(iface) {
			continue
		}

		if !hasAddressInNetwork(iface) {
			continue
		}

		return &iface, nil
	}
	return nil, fmt.Errorf("No usable interface found")
}
