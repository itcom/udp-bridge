//go:build windows

package main

import (
	"golang.org/x/sys/windows/registry"
)

func listSerialPorts() []string {
	var ports []string
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `HARDWARE\DEVICEMAP\SERIALCOMM`, registry.QUERY_VALUE)
	if err != nil {
		return ports
	}
	defer key.Close()

	names, err := key.ReadValueNames(-1)
	if err != nil {
		return ports
	}

	for _, name := range names {
		val, _, err := key.GetStringValue(name)
		if err == nil {
			ports = append(ports, val)
		}
	}
	return ports
}
