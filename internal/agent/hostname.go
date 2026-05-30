package agent

import "os"

func getHostname() string {
	h, err := os.Hostname()
	if err != nil {
		return "unknown"
	}
	return h
}

func strPtr(s string) *string { return &s }
