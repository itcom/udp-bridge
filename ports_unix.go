//go:build !windows

package main

import "path/filepath"

func listSerialPorts() []string {
	var ports []string
	for _, pattern := range []string{"/dev/cu.*", "/dev/tty.*"} {
		matches, _ := filepath.Glob(pattern)
		ports = append(ports, matches...)
	}
	return ports
}
