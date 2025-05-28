package storage

import "io"

var logImage []OnDiskEntry /*image of on-disk log*/

func initLogImage() {
	// empty
}

// LogSize returns number of entries in log.
func LogSize() int64 {
	return int64(len(logImage))
}

// LogTerm returns term number of last entry in log.
// Since `logImage` always initialized with `magicEntry`, it is always non empty.
func LogTerm() int64 {
	return logImage[LogSize()-1].LogTerm
}

func LogGetNth(n int64) (OnDiskEntry, error) {
	if LogSize() <= n {
		return OnDiskEntry{}, io.EOF
	}

	return logImage[n], nil
}

func logImageAppendEntry(entry OnDiskEntry) {
	logImage = append(logImage, entry)
}

func dropLogImageSinceNth(n int64) {
	logImage = logImage[:n]
}
