package main

import (
	"os"
)

func GetEnv(key, val string) string {
	if v := os.Getenv(key); v == "" {
		return val
	} else {
		return v
	}
}

func contains(arr []DayOfWeek, el DayOfWeek) bool {
	for _, s := range arr {
		if s == el {
			return true
		}
	}
	return false
}
