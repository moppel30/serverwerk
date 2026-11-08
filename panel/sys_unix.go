//go:build !windows

package main

import "syscall"

func diskSpace(p string) (uint64, uint64, bool) {
	var s syscall.Statfs_t
	if syscall.Statfs(p, &s) != nil {
		return 0, 0, false
	}
	return s.Blocks * uint64(s.Bsize), s.Bavail * uint64(s.Bsize), true
}
