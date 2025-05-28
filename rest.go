package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/igor-senin/raft-distributed-storage/net_subsys"
	"github.com/igor-senin/raft-distributed-storage/raft"
)

func HandleRoot(w http.ResponseWriter, r *http.Request) {
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

// Client requests
func HandleResource(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet: // = Read
		key, err := strconv.ParseInt(r.FormValue("key"), 10, 64)
		if err != nil {
			http.Error(w, "'key' must be provided in URL params", http.StatusBadRequest)
			return
		}

		w.Header().Set("Key", strconv.FormatInt(key, 10))

		value, ok := raft.ClientReadStub(key)
		if ok {
			w.Header().Set("Value", strconv.FormatInt(value, 10))
		} else {
			w.Header().Set("Value", "_")
		}

		w.WriteHeader(http.StatusOK)

	case http.MethodPost: // = Create
		key, err := strconv.ParseInt(r.FormValue("key"), 10, 64)
		if err != nil {
			http.Error(w, "'key' must be provided in URL params", http.StatusBadRequest)
			return
		}
		value, err := strconv.ParseInt(r.FormValue("value"), 10, 64)
		if err != nil {
			http.Error(w, "'value' must be provided in URL params", http.StatusBadRequest)
			return
		}

		w.Header().Set("Key", strconv.FormatInt(key, 10))

		err = raft.ClientCreateStub(key, value)
		if err != nil {
			var redirErr *raft.RedirectError
			if errors.As(err, &redirErr) {
				leaderIp, err := net_subsys.AddToIP(net_subsys.BaseIPAddr, redirErr.RedirectId)
				if err != nil {
					http.Error(w, "unknown error in AddToIP", http.StatusInternalServerError)
					return
				}
				w.Header().Set("LeaderIP", leaderIp)
				w.WriteHeader(http.StatusFound)
				return
			}

			var asyncErr *raft.AsyncError
			if errors.As(err, &asyncErr) {
				w.WriteHeader(http.StatusAccepted)
				return
			}

			http.Error(w, "unknown error in storage.Create", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	case http.MethodPut: // = Update
		key, err := strconv.ParseInt(r.FormValue("key"), 10, 64)
		if err != nil {
			http.Error(w, "'key' must be provided in URL params", http.StatusBadRequest)
			return
		}
		value, err := strconv.ParseInt(r.FormValue("value"), 10, 64)
		if err != nil {
			http.Error(w, "'value' must be provided in URL params", http.StatusBadRequest)
			return
		}

		w.Header().Set("Key", strconv.FormatInt(key, 10))

		err = raft.ClientUpdateStub(key, value)
		if err != nil {
			var redirErr *raft.RedirectError
			if errors.As(err, &redirErr) {
				leaderIp, err := net_subsys.AddToIP(net_subsys.BaseIPAddr, redirErr.RedirectId)
				if err != nil {
					http.Error(w, "unknown error in AddToIP", http.StatusInternalServerError)
					return
				}
				w.Header().Set("LeaderIP", leaderIp)
				w.WriteHeader(http.StatusFound)
				return
			}

			var asyncErr *raft.AsyncError
			if errors.As(err, &asyncErr) {
				w.WriteHeader(http.StatusAccepted)
				return
			}

			http.Error(w, "unknown error in storage.Update", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	case http.MethodDelete: // = Delete
		key, err := strconv.ParseInt(r.FormValue("key"), 10, 64)
		if err != nil {
			http.Error(w, "'key' must be provided in URL params", http.StatusBadRequest)
			return
		}

		w.Header().Set("Key", strconv.FormatInt(key, 10))

		err = raft.ClientDeleteStub(key)
		if err != nil {
			var redirErr *raft.RedirectError
			if errors.As(err, &redirErr) {
				leaderIp, err := net_subsys.AddToIP(net_subsys.BaseIPAddr, redirErr.RedirectId)
				if err != nil {
					http.Error(w, "unknown error in AddToIP", http.StatusInternalServerError)
					return
				}
				w.Header().Set("LeaderIP", leaderIp)
				w.WriteHeader(http.StatusFound)
				return
			}

			var asyncErr *raft.AsyncError
			if errors.As(err, &asyncErr) {
				w.WriteHeader(http.StatusAccepted)
				return
			}

			http.Error(w, "unknown error in storage.Delete", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

// --------------------------
// Cluster events
// --------------------------

// Handle
func HandleRPCAppendEntries(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed in AppendEntries", http.StatusMethodNotAllowed)
		return
	}

	var payload raft.AppendEntriesPayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	response := raft.AppendEntriesStub(payload)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}

func HandleRPCRequestVote(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST allowed in RequestVote", http.StatusMethodNotAllowed)
		return
	}

	var payload raft.RequestVotePayload
	err := json.NewDecoder(r.Body).Decode(&payload)
	if err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}

	response := raft.RequestVoteStub(payload)

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}
}
