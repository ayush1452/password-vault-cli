//go:build unix

package store

import (
	"os"
	"syscall"
)

// platformLock applies platform-specific locking (Unix flock)
func platformLock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

// platformUnlock releases platform-specific lock (Unix flock)
func platformUnlock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
