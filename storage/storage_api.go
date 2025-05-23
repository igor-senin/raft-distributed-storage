package storage

import (
	"log"
)

type (
	TKey   int64
	TValue int64
)

type OpType int8

const (
	OpCreate OpType = iota
	OpUpdate
	OpDelete
	NoOp
)

var opName = map[OpType]string{
	OpCreate: "create",
	OpUpdate: "update",
	OpDelete: "delete",
	NoOp:     "no operation",
}

func (op OpType) String() string {
	return opName[op]
}

type LogEntry struct {
	EntryType OpType `json:"type"`
	Key       TKey   `json:"key"`
	Value     TValue `json:"value"`
}

var storage map[TKey]TValue

// InitStorage is called is called once during the startup.
// It's responsible for initing the direct storage and log image subsystems.
func InitStorage() error {
	log.Println("[Init Storage]")

	if err := initDirectStorage(); err != nil {
		return err
	}
	storage = make(map[TKey]TValue)

	initLogImage()

	header, err := directReadHeader()
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] [Init Storage] header: %v\n", header)
	log.Println("[INFO] [Init Storage] log replay")
	err = replayLog(header.appliedIndex)
	if err != nil {
		return err
	}
	log.Println("[INFO] [Init Storage] success")

	return nil
}

// replayLog is called once during startup.
// It reads `appliedIdx` from header on disk
// and applies first `appliedIdx` entries in log to state machine.
func replayLog(appliedIdx int64) error {
	for i := range appliedIdx {
		entry, err := directReadNth(i)
		log.Printf("[DEBUG] [replay log] entry : %d ; entry : %v\n", i, entry)
		if err != nil {
			return err
		}

		logImageAppendEntry(entry)
		applyEntry(entry.Entry)

		nextToApplyIndex += 1
		log.Printf("[DEBUG] [replay log] nextToApplyIndex = %d\n", nextToApplyIndex)
	}
	return nil
}

// ReadRecord reads record with key 'key' from state machine.
func ReadRecord(key TKey) (TValue, bool) {
	value, ok := storage[key]
	return value, ok
}

// CreateRecord creates record (`key`, `value`).
// It is not applied to the state machine,
// but written to persistent log and can be later applied.
func CreateRecord(key TKey, value TValue, term int64) error {
	entry := LogEntry{
		EntryType: OpCreate,
		Key:       key,
		Value:     value,
	}
	on_disk := OnDiskEntry{
		Entry:    entry,
		LogTerm:  term,
		LogIndex: LogSize(),
	}
	if err := directAppendEntry(on_disk); err != nil {
		return err
	}
	logImageAppendEntry(on_disk)
	return nil
}

// UpdateRecord updates record (`key`, `value`).
// It is not applied to the state machine,
// but written to persistent log and can be later applied.
func UpdateRecord(key TKey, value TValue, term int64) error {
	entry := LogEntry{
		EntryType: OpUpdate,
		Key:       key,
		Value:     value,
	}
	on_disk := OnDiskEntry{
		Entry:    entry,
		LogTerm:  term,
		LogIndex: LogSize(),
	}
	if err := directAppendEntry(on_disk); err != nil {
		return err
	}
	logImageAppendEntry(on_disk)
	return nil
}

// DeleteRecord deletes record with key `key`.
// It is not applied to the state machine,
// but written to persistent log and can be later applied.
func DeleteRecord(key TKey, term int64) error {
	entry := LogEntry{
		EntryType: OpDelete,
		Key:       key,
		Value:     0,
	}
	on_disk := OnDiskEntry{
		Entry:    entry,
		LogTerm:  term,
		LogIndex: LogSize(),
	}
	if err := directAppendEntry(on_disk); err != nil {
		return err
	}
	logImageAppendEntry(on_disk)
	return nil
}

// HandleEntry is called for every LogEntry that came from leader.
func HandleEntry(entry LogEntry, term int64) error {
	key := TKey(entry.Key)
	value := TValue(entry.Key)

	var err error = nil
	switch entry.EntryType {
	case OpCreate:
		err = CreateRecord(key, value, term)
	case OpUpdate:
		err = UpdateRecord(key, value, term)
	case OpDelete:
		err = DeleteRecord(key, term)
	}

	return err
}

// applyEntry is inner function to apply `entry` to state machine.
func applyEntry(entry LogEntry) {
	key := TKey(entry.Key)
	value := TValue(entry.Key)

	switch entry.EntryType {
	case OpCreate:
		storage[key] = value
	case OpUpdate:
		storage[key] = value
	case OpDelete:
		delete(storage, key)
	}
}

// ApplyNextRecord reads next uncommited record from persistent log
// and applies it for state machine.
// Return value of io.EOF indicates that there are no uncommited records.
func ApplyNextRecord() error {
	return ApplyUntilNth(nextToApplyIndex)
}

// ApplyUntilNth applies entries from persistent log until index `n`.
func ApplyUntilNth(n int64) error {
	for ; nextToApplyIndex <= n; nextToApplyIndex += 1 {
		entry, err := directReadNth(nextToApplyIndex)
		if err != nil {
			return err
		}

		applyEntry(entry.Entry)

		err = entryApplied()
		if err != nil {
			return err
		}
	}

	return nil
}

// DropSinceNth drops all log records beginning with index n.
func DropSinceNth(n int64) error {
	dropLogImageSinceNth(n)
	return directDropSinceNth(n)
}
