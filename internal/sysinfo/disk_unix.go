//go:build linux || darwin

package sysinfo

import "syscall"

// diskFree returns the bytes available to unprivileged users on the
// filesystem holding dir (0 if unknown).
func diskFree(dir string) uint64 {
	var st syscall.Statfs_t
	if dir == "" || syscall.Statfs(dir, &st) != nil {
		return 0
	}
	return uint64(st.Bavail) * uint64(st.Bsize)
}
