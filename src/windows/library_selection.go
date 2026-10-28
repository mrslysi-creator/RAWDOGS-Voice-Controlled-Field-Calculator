package main

import "strings"

func containsLibraryLabel(labels []string, v string) bool {
	for _, x := range labels {
		if strings.EqualFold(x, v) {
			return true
		}
	}
	return false
}
