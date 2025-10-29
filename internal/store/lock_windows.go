//go:build windows

package store

import (
	"os"
	"syscall"
	"unsafe"
)

var (
	kernel32     = syscall.NewLazyDLL("kernel32.dll")
	lockFileEx   = kernel32.NewProc("LockFileEx")
	unlockFileEx = kernel32.NewProc("UnlockFileEx")
)

const (
	LOCKFILE_EXCLUSIVE_LOCK   = 0x00000002
	LOCKFILE_FAIL_IMMEDIATELY = 0x00000001
)

// platformLock applies platform-specific locking (Windows LockFileEx)
func platformLock(file *os.File) error {
	handle := syscall.Handle(file.Fd())

	var overlapped syscall.Overlapped
	ret, _, err := lockFileEx.Call(
		uintptr(handle),
		uintptr(LOCKFILE_EXCLUSIVE_LOCK|LOCKFILE_FAIL_IMMEDIATELY),
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return err
	}
	return nil
}

// platformUnlock releases platform-specific lock (Windows UnlockFileEx)
func platformUnlock(file *os.File) error {
	handle := syscall.Handle(file.Fd())

	var overlapped syscall.Overlapped
	ret, _, err := unlockFileEx.Call(
		uintptr(handle),
		uintptr(0),
		uintptr(1),
		uintptr(0),
		uintptr(unsafe.Pointer(&overlapped)),
	)

	if ret == 0 {
		return err
	}
	return nil
}
