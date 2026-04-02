//go:build !darwin && !linux

package jsbridge

// platformStatTimes — unsupported platform, fall back to mtime.
func platformStatTimes(sys any) (atimeMs, ctimeMs, birthtimeMs int64, ok bool) {
	return 0, 0, 0, false
}

// platformStatFields — unsupported platform, use defaults.
func platformStatFields(sys any) (uid, gid uint32, dev, ino uint64, nlink uint16, rdev uint64, blksize int32, blocks int64, ok bool) {
	return 0, 0, 0, 0, 0, 0, 0, 0, false
}
