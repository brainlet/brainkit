//go:build linux

package jsbridge

import (
	"syscall"
	"time"
)

// platformStatTimes extracts atime, ctime from syscall.Stat_t on linux.
// Linux has no birthtime — returns ctime as fallback.
func platformStatTimes(sys any) (atimeMs, ctimeMs, birthtimeMs int64, ok bool) {
	st, is := sys.(*syscall.Stat_t)
	if !is {
		return 0, 0, 0, false
	}
	atime := time.Unix(st.Atim.Sec, st.Atim.Nsec).UnixMilli()
	ctime := time.Unix(st.Ctim.Sec, st.Ctim.Nsec).UnixMilli()
	return atime, ctime, ctime, true // birthtime = ctime on linux
}

// platformStatFields extracts uid, gid, dev, ino, nlink, rdev, blksize, blocks from Stat_t on linux.
func platformStatFields(sys any) (uid, gid uint32, dev, ino uint64, nlink uint16, rdev uint64, blksize int32, blocks int64, ok bool) {
	st, is := sys.(*syscall.Stat_t)
	if !is {
		return 0, 0, 0, 0, 0, 0, 0, 0, false
	}
	return st.Uid, st.Gid, st.Dev, st.Ino, uint16(st.Nlink), st.Rdev, st.Blksize, st.Blocks, true
}
