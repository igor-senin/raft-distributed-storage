package storage

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

var (
	Filename string = "log_raft.bin"
	file     *os.File
)

var magicEntry OnDiskEntry = OnDiskEntry{
	LogTerm:  -15151515,  /*magic numbers*/
	LogIndex: -7952812,   /*magic numbers*/
	Entry:    LogEntry{}, /*empty entry*/
}

var nextToApplyIndex int64 /*index of next record to be applied*/

type OnDiskHeader struct {
	version      int64 /**/
	appliedIndex int64 /*index of last applied record*/
}

type OnDiskEntry struct {
	LogTerm  int64    /*term number of this entry*/
	LogIndex int64    /*index in log of this entry*/
	Entry    LogEntry /*payload*/
}

func CurrentAppliedIndex() int64 {
	return nextToApplyIndex
}

/*DirectReadNth TODO: descr*/
func directReadNth(n int64) (OnDiskEntry, error) {
	if file == nil {
		return OnDiskEntry{}, errors.New("file is not initialized")
	}
	if LogSize() < n {
		return OnDiskEntry{}, io.EOF
	}
	return fileReadNth(file, n)
}

func entryApplied() error {
	nextToApplyIndex += 1

	new_header := OnDiskHeader{
		version:      1,
		appliedIndex: nextToApplyIndex,
	}
	err := directUpdateHeader(new_header)
	if err != nil {
		return err
	}

	return nil
}

// initDirectStorage inits physical storage, i.e. real file.
// Can be called on any startup (first or after crash).
func initDirectStorage() error {
	file_ptr, err := initFile()
	if err != nil {
		return nil
	}

	file = file_ptr

	nextToApplyIndex = 0

	return nil
}

/*directAppendEntry TODO: description*/
func directAppendEntry(entry OnDiskEntry) error {
	if file == nil {
		return errors.New("file is not initialized")
	}

	err := fileAppendEntry(file, entry)
	if err != nil {
		return err
	}

	return nil
}

func directUpdateHeader(header OnDiskHeader) error {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.LittleEndian, header)
	if err != nil {
		return err
	}
	data := buf.Bytes()
	_, err = unix.Pwrite(int(file.Fd()), data, 0)
	if err != nil {
		return err
	}

	return nil
}

func directReadHeader() (OnDiskHeader, error) {
	buf := make([]byte, headerSize())
	_, err := unix.Pread(int(file.Fd()), buf, 0)
	if err != nil {
		return OnDiskHeader{}, err
	}

	var header OnDiskHeader
	err = binary.Read(bytes.NewReader(buf), binary.LittleEndian, &header)
	if err != nil {
		return OnDiskHeader{}, err
	}

	return header, nil
}

func fileAppendEntry(file_ptr *os.File, entry OnDiskEntry) error {
	return binary.Write(file_ptr, binary.LittleEndian, entry)
}

func initFile() (*os.File, error) {
	new_file_ptr, err := os.OpenFile(
		Filename,
		os.O_CREATE|os.O_RDWR|os.O_APPEND,
		0o644,
	)
	if err != nil {
		return nil, err
	}

	err = ensureMagicValid(new_file_ptr)
	if err != nil {
		return nil, err
	}

	_, err = new_file_ptr.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, err
	}

	return new_file_ptr, nil
}

func ensureMagicValid(file_ptr *os.File) error {
	if !isMagicValid(file_ptr) {
		if err := clearAndInitFile(file_ptr); err != nil {
			return err
		}
	}
	return nil
}

func clearAndInitFile(file_ptr *os.File) error {
	var err error

	err = file_ptr.Truncate(0)
	if err != nil {
		return err
	}

	header := OnDiskHeader{
		version:      1,
		appliedIndex: 1,
	}
	err = binary.Write(file_ptr, binary.LittleEndian, header)
	if err != nil {
		return err
	}
	err = binary.Write(file_ptr, binary.LittleEndian, magicEntry)
	if err != nil {
		return err
	}

	return nil
}

func isMagicValid(file_ptr *os.File) bool {
	firstEntry, err := fileReadNth(file_ptr, 0)
	if err != nil ||
		firstEntry.LogIndex != magicEntry.LogIndex ||
		firstEntry.LogTerm != magicEntry.LogTerm {
		return false
	}

	return true
}

func fileReadNth(file_ptr *os.File, n int64) (OnDiskEntry, error) {
	buf := make([]byte, entrySize())
	_, err := unix.Pread(int(file_ptr.Fd()), buf, getNthOffset(n))
	if err != nil {
		return OnDiskEntry{}, err
	}

	var entry OnDiskEntry
	err = binary.Read(bytes.NewReader(buf), binary.LittleEndian, &entry)
	if err != nil {
		return OnDiskEntry{}, err
	}

	return entry, nil
}

func directDropSinceNth(n int64) error {
	err := file.Truncate(getNthOffset(n))
	if err != nil {
		return err
	}

	return nil
}

func entrySize() int64 {
	return int64(binary.Size(OnDiskEntry{}))
}

func headerSize() int64 {
	return int64(binary.Size(OnDiskHeader{}))
}

func getNthOffset(n int64) int64 {
	return n*entrySize() + headerSize()
}
