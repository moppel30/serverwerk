//go:build windows

package main

func diskSpace(p string) (uint64, uint64, bool) { return 0, 0, false }
