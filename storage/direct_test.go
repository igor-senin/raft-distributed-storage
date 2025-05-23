package storage

import (
	"fmt"
	"log"
	"os"
	"testing"
)

var test_file *os.File

func TestMain(m *testing.M) {
	fmt.Println("entrySize: {}", entrySize())
	fmt.Println("headerSize: {}", headerSize())
	fmt.Println("nthoffset(1): {}", getNthOffset(1))

	fmt.Println("Creating test file...")

	file_ptr, err := initFile()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("OK")

	test_file = file_ptr

	code := m.Run()
	log.Println("code : {}", code)

	test_file.Close()
	os.Remove(Filename)
	os.Exit(code)
}

func TestFileWriteRead(t *testing.T) {
	if err := clearAndInitFile(test_file); err != nil {
		t.Fatal(err)
	}

	var err error

	t.Log("Running fileAppendEntry...")

	err = fileAppendEntry(test_file, magicEntry)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("Running fileReadNth...")

	entry, err := fileReadNth(test_file, 1)
	if err != nil {
		t.Fatal(err)
	}
	if entry != magicEntry {
		t.Fatalf("entry = %v != magicEntry = %v", entry, magicEntry)
	}
	t.Log("OK")
}

func TestNWritesNReads(t *testing.T) {
	if err := clearAndInitFile(test_file); err != nil {
		t.Fatal(err)
	}

	var err error

	t.Log("Running fileAppendEntry...")

	for i := range 100 {
		entry := magicEntry
		entry.Entry.Key = TKey(i)
		entry.Entry.Value = TValue(2 * i)
		err = fileAppendEntry(test_file, entry)
		if err != nil {
			t.Fatal(err)
		}
	}

	t.Log("Running fileReadNth...")

	for i := range 100 {
		entry, err := fileReadNth(test_file, int64(i)+1)
		if err != nil {
			t.Fatal(err)
		}
		expected := magicEntry
		expected.Entry.Key = TKey(i)
		expected.Entry.Value = TValue(i * 2)
		if entry != expected {
			t.Fatalf("entry = %v != expected = %v", entry, expected)
		}
	}
	t.Log("OK")
}
