package raft

import "time"

var (
	TermNumber  int64
	VotedFor    int64
	CommitIndex int64 // index of highest log entry known to be committed
	LogSize     int64 // amount of entries in log
)

// Server state
var (
	NextIndex  []int64 // for each node index of the next log entry to send to that server
	MatchIndex []int64 // for each node index of the log entry server knows is commited on that node
)

var (
	IsLeader    bool
	ClusterSize int
	Idx         int
)

func MainCycle() {
	var timer time.Timer

	select {
	case <-timer.C:
	}
}
