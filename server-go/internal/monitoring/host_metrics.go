package monitoring

import "syscall"

// diskUsage reports real filesystem usage for path using statfs. It is
// dependency-free and works on the macOS dev host and the Linux NAS alike, so
// the dev collector can surface a genuinely live disk metric instead of a
// hardcoded value. ok is false when the syscall fails.
func hostDiskUsage(path string) (usedPercent float64, usedGB, totalGB float64, ok bool) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0, 0, false
	}
	blockSize := uint64(st.Bsize)
	total := st.Blocks * blockSize
	free := st.Bavail * blockSize
	if total == 0 {
		return 0, 0, 0, false
	}
	used := total - free
	const gib = 1024 * 1024 * 1024
	return float64(used) / float64(total) * 100, float64(used) / gib, float64(total) / gib, true
}
