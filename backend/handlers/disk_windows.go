//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd

package handlers

import (
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

// platformDiskUsage reports real filesystem usage on Windows development hosts
// so the dashboard is not showing a fabricated number while running locally.
// In production the binary always runs on Linux inside the container.
func platformDiskUsage(path string) (total uint64, used uint64) {
	abs, err := filepath.Abs(path)
	if err != nil {
		abs = path
	}

	// Resolve the volume root (e.g. "C:\"). GetDiskFreeSpaceEx needs a root path.
	vol := filepath.VolumeName(abs)
	if vol == "" {
		return 0, 0
	}
	root := vol + string(os.PathSeparator)

	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	proc := kernel32.NewProc("GetDiskFreeSpaceExW")

	rootPtr, err := syscall.UTF16PtrFromString(root)
	if err != nil {
		return 0, 0
	}

	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64
	r, _, _ := proc.Call(
		uintptr(unsafe.Pointer(rootPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalNumberOfBytes)),
		uintptr(unsafe.Pointer(&totalNumberOfFreeBytes)),
	)
	if r == 0 {
		return 0, 0
	}

	total = totalNumberOfBytes
	used = totalNumberOfBytes - totalNumberOfFreeBytes
	return total, used
}
