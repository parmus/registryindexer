package utils

import "strings"

// MinInt returns the minimum of two integers (why is this not a standard function?)
func MinInt(x, y int) int {
	if x > y {
		return y
	}
	return x
}

// HasAnyPrefix works similar to [strings.HasPrefix] but with a list of possible prefixes
func HasAnyPrefix(prefixes []string, value string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(value, prefix) {
			return true
		}
	}
	return false
}
