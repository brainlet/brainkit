//go:build darwin

package jsbridge

import (
	"syscall"
	"time"
)

// platformStatTimes extracts atime, ctime, birthtime from syscall.Stat_t on darwin.
func platformStatTimes(sys any) (atimeMs, ctimeMs, birthtimeMs int64, ok bool) {
	st, is := sys.(*syscall.Stat_t)
	if !is {
		return 0, 0, 0, false
	}
	return time.Unix(st.Atimespec.Sec, st.Atimespec.Nsec).UnixMilli(),
		time.Unix(st.Ctimespec.Sec, st.Ctimespec.Nsec).UnixMilli(),
		time.Unix(st.Birthtimespec.Sec, st.Birthtimespec.Nsec).UnixMilli(),
		true
}

// platformStatFields extracts uid, gid, dev, ino, nlink, rdev, blksize, blocks from Stat_t on darwin.
func platformStatFields(sys any) (uid, gid uint32, dev, ino uint64, nlink uint16, rdev uint64, blksize int32, blocks int64, ok bool) {
	st, is := sys.(*syscall.Stat_t)
	if !is {
		return 0, 0, 0, 0, 0, 0, 0, 0, false
	}
	return st.Uid, st.Gid, uint64(st.Dev), st.Ino, st.Nlink, uint64(st.Rdev), st.Blksize, st.Blocks, true
}
