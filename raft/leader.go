package raft

import (
	"encoding/json"
	"log"
	"time"

	"github.com/igor-senin/raft-distributed-storage/net_subsys"
	"github.com/igor-senin/raft-distributed-storage/storage"
)

// AppendEntries is called by leader,
// either on client request, or on timer timeout.
func AppendEntries(
	dstId int64,
	entries []LogEntry,
	prevLogIndex,
	prevLogTerm int64,
	timeout time.Duration,
) (AppendEntriesResponse, error) {
	payload := AppendEntriesPayload{
		Term:         TermNumber,
		LeaderId:     Idx,
		PrevLogIndex: prevLogIndex,
		PrevLogTerm:  prevLogTerm,
		Entries:      entries,
		LeaderCommit: CommitIndex,
	}
	appendEntriesPath := "rpc/append_entries"
	resp, err := net_subsys.SendToId(dstId, appendEntriesPath, payload, timeout)
	if err != nil {
		return AppendEntriesResponse{}, err
	}

	defer resp.Body.Close() // TODO: для чего?

	log.Println("[DEBUG] [AppendEntries] status =", resp.Status)

	var responsePayload AppendEntriesResponse
	err = json.NewDecoder(resp.Body).Decode(&responsePayload)
	if err != nil {
		return AppendEntriesResponse{}, err
	}

	log.Println("[DEBUG] [AppendEntries] response :", responsePayload)

	return responsePayload, nil
}

// RequestVote is called by candidate for every other node.
func RequestVote(dstId int64, timeout time.Duration) (RequestVoteResponse, error) {
	payload := RequestVotePayload{
		Term:         TermNumber,
		CandidateId:  Idx,
		LastLogIndex: storage.LogSize(),
		LastLogTerm:  storage.LogTerm(),
	}
	requestVotePath := "rpc/request_vote"
	resp, err := net_subsys.SendToId(dstId, requestVotePath, payload, timeout)
	if err != nil {
		return RequestVoteResponse{}, err
	}

	defer resp.Body.Close() // TODO: для чего?

	log.Println("[DEBUG] [RequestVote] status =", resp.Status)

	var responsePayload RequestVoteResponse
	err = json.NewDecoder(resp.Body).Decode(&responsePayload)
	if err != nil {
		return RequestVoteResponse{}, err
	}

	log.Println("[DEBUG] [RequestVote] response :", responsePayload)

	return responsePayload, nil
}
