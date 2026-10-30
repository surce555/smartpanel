//go:build linux || darwin || freebsd || openbsd || netbsd

package handlers

import (
	"log"
	"syscall"
)

// platformDiskUsage reports real filesystem usage for the given mount point.
//
// This used to be a hardcoded 100GB / 25GB placeholder in system.go, so the
// dashboard always displayed a fake disk bar.
func platformDiskUsage(path string) (total uint64, used uint64) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		log.Printf("Warning: statfs %s failed: %v", path, err)
		return 0, 0
	}

	bsize := uint64(stat.Bsize)
	total = stat.Blocks * bsize

	// Bfree counts blocks free for root; Bavail counts blocks free for a
	// non-privileged user. Report usage the way a NAS user expects it.
	if stat.Bavail <= stat.Blocks {
		used = (stat.Blocks - stat.Bavail) * bsize
	} else {
		used = (stat.Blocks - stat.Bfree) * bsize
	}
	if used > total {
		used = total
	}
	return total, used
}
