//go:build !linux && !darwin

package sysinfo

func diskFree(string) uint64 { return 0 }
