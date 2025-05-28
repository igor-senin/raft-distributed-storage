package raft

import (
	"context"
	"fmt"
	"io"
	"log"

	"github.com/igor-senin/raft-distributed-storage/storage"
)

type AppendEntriesPayload struct {
	Term         int64      `json:"term"`         /*leader's term*/
	LeaderId     int64      `json:"leaderId"`     /*so follower can redirect clients*/
	PrevLogIndex int64      `json:"prevLogIndex"` /*index of log entry immediately preceding new ones*/
	PrevLogTerm  int64      `json:"prevLogTerm"`  /*term of prevLogIndex entry*/
	Entries      []LogEntry `json:"entries"`      /*log entries to store*/
	LeaderCommit int64      `json:"leaderCommit"` /*leader's commitIndex*/
}

type AppendEntriesResponse struct {
	Term     int64 `json:"term"`     /*for leader to update itself*/
	Accepted bool  `json:"accepted"` /*true if follower contained entry matching prevLogIndex and prevLogTerm*/
}

type RequestVotePayload struct {
	Term         int64 `json:"term"`         /*candidate's term*/
	CandidateId  int64 `json:"candidateId"`  /*candidate requesting vote*/
	LastLogIndex int64 `json:"lastLogIndex"` /*index of candidate's last log entry*/
	LastLogTerm  int64 `json:"lastLogTerm"`  /*term of candidate's last log entry*/
}

type RequestVoteResponse struct {
	Term        int64 `json:"term"`        /*for candidate to update itself*/
	VoteGranted bool  `json:"voteGranted"` /*true means candidate received vote*/
}

// AppendEntriesStub is called by REST handle for rpc/append_entries.
func AppendEntriesStub(payload AppendEntriesPayload) AppendEntriesResponse {
	var response AppendEntriesResponse
	response.Term = TermNumber
	response.Accepted = false

	leaderId := payload.LeaderId
	leadersTerm := payload.Term
	prevLogIndex := payload.PrevLogIndex
	prevLogTerm := payload.PrevLogTerm
	leaderCommit := payload.LeaderCommit

	if leadersTerm < TermNumber {
		return response
	}

	// Update own term number and known leader id.
	if leadersTerm > TermNumber {
		TermNumber = leadersTerm
		VotedFor = -1
	}
	LeaderId = leaderId

	// Leader is alive.
	// Signal event loop to reset timer.
	AEChannel <- true

	// Fast path if it was just ping from leader.
	if len(payload.Entries) == 0 {
		response.Accepted = true
		return response
	}

	// Check if log entry at `prevLogIndex` index has the same term.
	// If it does, then this entry and all before are the same at leader and follower.
	// Otherwise, we must discard all entries beginning with `prevLogIndex`
	// and wait for leader to sync us.
	disk_entry, err := storage.LogGetNth(prevLogIndex)
	// Don't have `prevLogIndex` in log.
	if err == io.EOF {
		return response
	}
	if err != nil {
		log.Fatal(err)
	}

	// dist_entry.LogIndex == prevLogIndex

	if disk_entry.LogTerm != prevLogTerm {
		storage.DropSinceNth(prevLogIndex)
		// Need server to resend `prevLogIndex` entry.
		return response
	}

	// We accepted leader's entries.
	response.Accepted = true

	for i, entry := range payload.Entries {
		currIndex := prevLogIndex + 1 + int64(i)
		currEntry, err := storage.LogGetNth(currIndex)
		if err != nil && err != io.EOF {
			log.Fatal(err)
		}
		// If already have this entry, skip
		if err == nil && currEntry.LogTerm == entry.Term {
			continue
		}
		if err != io.EOF {
			storage.DropSinceNth(currIndex)
		}

		// ... or apply this entry.
		_ = storage.HandleEntry(entry.LogEntry, entry.Term)
		// TODO: handle err above
	}

	// Logic for follower to commit entries.
	if leaderCommit > CommitIndex {
		CommitIndex = min(leaderCommit, storage.LogSize())
		err := storage.ApplyUntilNth(CommitIndex)
		if err != nil {
			log.Fatal(err)
		}
	}

	return response
}

// RequestVoteStub is called by REST handler for rpc/request_vote.
func RequestVoteStub(payload RequestVotePayload) RequestVoteResponse {
	var response RequestVoteResponse
	response.Term = TermNumber
	response.VoteGranted = false

	candidatesTerm := payload.Term
	candidateId := payload.CandidateId
	candidatesLastLogIdx := payload.LastLogIndex
	candidatesLastLogTerm := payload.LastLogTerm

	if IsLeader ||
		TermNumber > candidatesTerm ||
		(TermNumber == candidatesTerm && VotedFor != candidateId && VotedFor != -1) {
		return response
	}

	// At this point:
	// not leader && termNumber <= candidatesTerm && didn't vote for anyone else in this term.

	// Check if candidate is fresh enough.
	if checkCandidateLogUpToDate(candidatesLastLogIdx, candidatesLastLogTerm) {
		log.Printf("[DEBUG] [RequestVoteStub] candidates log is up to date \n")
		// Update current term number.
		// If there are absent log entries, future leaders will catch up.
		TermNumber = candidatesTerm
		VotedFor = candidateId
		response.VoteGranted = true
	} else {
		log.Printf("[DEBUG] [RequestVoteStub] BAAAAAD!!!!!!!!!! CANDIDATE BAAAAAAAAAAAD!!!!! \n")
	}

	RVChannel <- true

	return response
}

func checkCandidateLogUpToDate(candidateLogSize, candidateLogTerm int64) bool {
	log.Printf("[DEBUG] [checkCandidateLogUpToDate] candidateLogSize = %d ; candidatesTerm = %d\n", candidateLogSize, candidateLogTerm)
	myLogSize := storage.LogSize()
	myLogTerm := storage.LogTerm()
	log.Printf("[DEBUG] [checkCandidateLogUpToDate] myLogSize = %d ; myLogTerm = %d\n", myLogSize, myLogTerm)

	if myLogTerm != candidateLogTerm {
		return myLogTerm < candidateLogTerm
	}

	return myLogSize <= candidateLogSize
}

// serveOneReplica sends the straggler replica `idx` new records.
// Writes to channel `confirmChan`, if successfully updated replica.
// Returns:
//
//	true, if we still can be leader
//	false, if we received response with bigger term number than ours.
func serveOneReplica(idx int64, confirmChan chan bool) bool {
	entries := []LogEntry{}
	log.Printf("[DEBUG] [broadcastAppendEntry] i: %d ; NextIndex[idx]: %d \n", idx, NextIndex[idx])
	for n := NextIndex[idx]; n < storage.LogSize(); n += 1 {
		onDisk, err := storage.LogGetNth(n)
		if err != nil {
			log.Printf("[INFO] [broadcastAppendEntry] error while reading %d'th record from log ; i: %d ; error: %v\n", n, idx, err)
			return true
		}
		newEntry := LogEntry{
			LogEntry: onDisk.Entry,
			Term:     onDisk.LogTerm,
		}
		entries = append(entries, newEntry)
	}

	for {
		prevLogIndex := NextIndex[idx] - 1
		onDiskPrev, err := storage.LogGetNth(prevLogIndex)
		if err != nil {
			log.Printf("[INFO] [broadcastAppendEntry] error while reading %d'th record from log ; i: %d ; error: %v\n", prevLogIndex, idx, err)
			return true
		}
		prevLogTerm := onDiskPrev.LogTerm
		resp, err := AppendEntries(int64(idx), entries, prevLogIndex, prevLogTerm, AppendEntriesTO)
		if err != nil {
			log.Printf("[INFO] [broadcastAppendEntry] i: %d ; error: %v\n", idx, err)
			return true
		}
		if resp.Term > TermNumber {
			log.Printf("[INFO] [broadcastAppendEntry] i: %d has bigger term: %d > %d\n", idx, resp.Term, TermNumber)
			return false
		}

		if resp.Accepted {
			confirmChan <- true
			NextIndex[idx] = storage.LogSize()
			return true
		}

		// If not accepted, continue sending entries.
		NextIndex[idx] -= 1

		prevEntry := LogEntry{
			LogEntry: onDiskPrev.Entry,
			Term:     prevLogTerm,
		}
		entries = append([]LogEntry{prevEntry}, entries...)
	}
}

type AsyncError struct{} // `AsyncError` tells client that cluster is in bad condition now and returns code 202.

func (e *AsyncError) Error() string {
	return "asynchronous commit of entry"
}

func broadcastAppendEntry() error {
	confirmsChan := make(chan bool, ClusterSize)

	for i := range ClusterSize {
		if int64(i) == Idx {
			continue
		}

		go serveOneReplica(i, confirmsChan)
	}

	timeout := AppendEntriesTO
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for range QuorumSize {
		select {
		case <-confirmsChan:
			continue
		case <-ctx.Done():
			return &AsyncError{}
		}
	}

	// Quorum has been assembled. Can commit it now.
	CommitIndex += 1
	if err := storage.ApplyNextRecord(); err != nil {
		log.Printf("[INFO] [broadcastAppendEntry] error occured in ApplyNextRecord: %v \n", err)
		return err
	}

	return nil
}

// ----------------------
// Client API calls
// ----------------------

// RedirectError describes error, that is used to tell client who is current leader.
type RedirectError struct {
	RedirectId int64
}

func (e *RedirectError) Error() string {
	return fmt.Sprintf("redirect to id: %d", e.RedirectId)
}

func ClientReadStub(key int64) (int64, bool) {
	val, ok := storage.ReadRecord(storage.TKey(key))

	return int64(val), ok
}

func ClientCreateStub(key int64, value int64) error {
	if !IsLeader {
		return &RedirectError{RedirectId: LeaderId}
	}

	if err := storage.CreateRecord(storage.TKey(key), storage.TValue(value), TermNumber); err != nil {
		return err
	}

	return broadcastAppendEntry()
}

func ClientUpdateStub(key int64, value int64) error {
	if !IsLeader {
		return &RedirectError{RedirectId: LeaderId}
	}

	if err := storage.UpdateRecord(storage.TKey(key), storage.TValue(value), TermNumber); err != nil {
		return err
	}

	return broadcastAppendEntry()
}

func ClientDeleteStub(key int64) error {
	if !IsLeader {
		return &RedirectError{RedirectId: LeaderId}
	}

	if err := storage.DeleteRecord(storage.TKey(key), TermNumber); err != nil {
		return err
	}

	return broadcastAppendEntry()
}
