package raft

import (
	"net/netip"

	"github.com/igor-senin/raft-distributed-storage/storage"
)

var (
	IPAddr     netip.Addr
	BaseIPAddr netip.Addr
)

type LogEntry struct {
	entryType int
	key       storage.TKey
	value     storage.TValue
}

func AppendEntriesStub(
	term, /*leader's term*/
	leaderId, /*so follower can redirect clients*/
	prevLogIndex, /*index of log entry immediately preceding new ones*/
	prevLogTerm int64, /*term of prevLogIndex entry*/
	entries []LogEntry, /*log entries to store*/
	leaderCommit int64, /*leader's commitIndex*/
) (int64, bool) {
	if TermNumber > term {
		return TermNumber, false
	}

	return TermNumber, true
}

func RequestVoteStub(
	term, /*candidate's term*/
	candidateId, /*candidate requesting vote*/
	lastLogIndex, /*index of candidate's last log entry*/
	lastLogTerm int64, /*term of candidate's last log entry*/
) (int64, bool) {
	if TermNumber > term {
		return TermNumber, false
	}

	// if raft.VotedFor == 0 ||  {
	// 	return raft.TermNumber, true
	// }
	return 0, true
}
