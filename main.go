package main

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/igor-senin/raft-distributed-storage/cli"
	"github.com/igor-senin/raft-distributed-storage/storage"
)

func writeRecord(f *os.File, key, value uint64) error {
	buf := make([]byte, 16)
	binary.LittleEndian.PutUint64(buf[0:8], key)
	binary.LittleEndian.PutUint64(buf[8:16], value)
	_, err := f.Write(buf)
	return err
}

func readLastRecord(f *os.File) (uint64, uint64, error) {
	info, err := f.Stat()
	if err != nil {
		return 0, 0, err
	}
	if info.Size() < 16 {
		return 0, 0, fmt.Errorf("file too small")
	}

	buf := make([]byte, 16)
	_, err = f.ReadAt(buf, info.Size()-16)
	if err != nil {
		return 0, 0, err
	}

	key := binary.LittleEndian.Uint64(buf[:8])
	val := binary.LittleEndian.Uint64(buf[8:])
	return key, val, nil
}

func main() {
	// f, err := os.OpenFile("log_queue", os.O_CREATE|os.O_RDWR, 0o644)
	// if err != nil {
	// 	panic(err)
	// }
	// defer f.Close()
	//
	// _ = writeRecord(f, 123, 456)
	// _ = writeRecord(f, 789, 1011)
	//
	// key, val, err := readLastRecord(f)
	// if err != nil {
	// 	panic(err)
	// }
	// fmt.Printf("Last record: key=%d, val=%d\n", key, val)

	var err error = nil

	err = cli.InitCommandLine()
	if err != nil {
		log.Fatal(err)
	}

	err = storage.InitStorage()
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/resource", handleResource)

	addr := "0.0.0.0:8786"
	fmt.Println("Listening on", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Hello from GET!")

	case http.MethodPost:
		var data map[string]any
		err := json.NewDecoder(r.Body).Decode(&data)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
		}

		response := map[string]any{
			"received": data,
			"status":   "POST success",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func handleResource(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		// Read
		fmt.Println("GET params were: ", r.URL.Query())
		fmt.Println("GET params[key] = ", r.FormValue("key"))

		key, err := strconv.Atoi(r.FormValue("key"))
		if err != nil {
			http.Error(w, "Invalid key", http.StatusBadRequest)
			return
		}

		w.Header().Set("Key", strconv.Itoa(key))

		value, ok := storage.ReadRecord(storage.TKey(key))
		if ok {
			w.Header().Set("Value", strconv.Itoa(int(value)))
		} else {
			w.Header().Set("Value", "_")
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Hello from GET!")

	case http.MethodPost:
		// Create
		fmt.Println("POST params were: ", r.URL.Query())
		fmt.Println("POST params[key] = ", r.FormValue("key"))
		fmt.Println("POST params[value] = ", r.FormValue("value"))

		key, err := strconv.Atoi(r.FormValue("key"))
		if err != nil {
			http.Error(w, "'key' must be provided in URL params", http.StatusBadRequest)
			return
		}
		value, err := strconv.Atoi(r.FormValue("value"))
		if err != nil {
			http.Error(w, "'value' must be provided in URL params", http.StatusBadRequest)
			return
		}

		w.Header().Set("Key", strconv.Itoa(key))
		err = storage.CreateRecord(storage.TKey(key), storage.TValue(value))
		if err != nil {
			http.Error(w, "Cannot create record", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Hello from POST!")

	case http.MethodPut:
		// Update
		fmt.Println("PUT params were: ", r.URL.Query())
		fmt.Println("PUT params[key] = ", r.FormValue("key"))
		fmt.Println("PUT params[value] = ", r.FormValue("value"))

		key, err := strconv.Atoi(r.FormValue("key"))
		if err != nil {
			http.Error(w, "'key' must be provided in URL params", http.StatusBadRequest)
			return
		}
		value, err := strconv.Atoi(r.FormValue("value"))
		if err != nil {
			http.Error(w, "'value' must be provided in URL params", http.StatusBadRequest)
			return
		}

		w.Header().Set("Key", strconv.Itoa(key))

		old_value, ok := storage.UpdateRecord(storage.TKey(key), storage.TValue(value))
		if ok {
			w.Header().Set("Old-value", strconv.Itoa(int(old_value)))
		} else {
			w.Header().Set("Old-value", "_")
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Hello from PUT!")

	case http.MethodDelete:
		// Delete
		fmt.Println("DELETE params were: ", r.URL.Query())
		fmt.Println("DELETE params[key] = ", r.FormValue("key"))

		key, err := strconv.Atoi(r.FormValue("key"))
		if err != nil {
			http.Error(w, "'key' must be provided in URL params", http.StatusBadRequest)
			return
		}

		old_value, ok := storage.DeleteRecord(storage.TKey(key))
		if ok {
			w.Header().Set("Old-value", strconv.Itoa(int(old_value)))
		} else {
			w.Header().Set("Old-value", "_")
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Hello from DELETE!")

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}
