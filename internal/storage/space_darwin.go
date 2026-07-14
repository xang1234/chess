package storage

import "syscall"

const importSafetyMargin = 512 * 1024 * 1024

func RequiredImportBytes(compressedSize int64) uint64 {
	if compressedSize < 0 {
		compressedSize = 0
	}
	return uint64(compressedSize)*10 + importSafetyMargin
}

func AvailableBytes(path string) (uint64, error) {
	var stats syscall.Statfs_t
	if err := syscall.Statfs(path, &stats); err != nil {
		return 0, err
	}
	return uint64(stats.Bavail) * uint64(stats.Bsize), nil
}
