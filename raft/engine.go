package raft

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/igor-senin/raft-distributed-storage/storage"
)

// Same as storage.LogEntry, but with `term` for client to know term number of entry.
type LogEntry struct {
	storage.LogEntry       // embed LogEntry from storage
	Term             int64 `json:"term"` // term number of this entry
}

// For client API requests.
type FromClientLogEntry storage.LogEntry

var (
	TermNumber  int64 // latest term server has seen
	VotedFor    int64 // id of candidate that received vote in current term; -1 if no candidate
	LeaderId    int64 // id of last known leader
	CommitIndex int64 // index of highest log entry known to be committed
)

// Leader state
var (
	NextIndex  []int64 // for each node index of the next log entry to send to that server
	MatchIndex []int64 // for each node index of the log entry server knows is commited on that node
)

var (
	IsLeader    bool
	ClusterSize int64
	QuorumSize  int64
	Idx         int64
)

// channels
var (
	AEChannel        chan bool
	RVChannel        chan bool
	ClientAPIChannel chan FromClientLogEntry
	GrantsChannel    chan RequestVoteResponse
)

const (
	IdleTO          = 5 * time.Second        // time after which we start new elections, if old leader had not ping us
	RequestVoteTO   = 1 * time.Second        // time that we wait for grants in `runElections`
	AppendEntriesTO = 500 * time.Millisecond // time that leader waits for other hosts to confirm they appended entries
	HeartbitTO      = 2 * time.Second        // time after which leader sends heartbit
)

func initRaft() {
	AEChannel = make(chan bool)
	RVChannel = make(chan bool)
	ClientAPIChannel = make(chan FromClientLogEntry)
	GrantsChannel = make(chan RequestVoteResponse, ClusterSize)

	NextIndex = make([]int64, ClusterSize)
	for i := range ClusterSize {
		NextIndex[i] = 1
	}

	QuorumSize = ClusterSize/2 + 1
	TermNumber = 0
	VotedFor = -1
	CommitIndex = 0
}

func RunRaft() {
	initRaft()

	for {
		if IsLeader {
			broadcastHeartbeat()

			LeaderRoutine()

			// Exited, because other hosts no longer consider us as leader,
			// and we received heartbit from new leader with greater `TermNumber`.
		} else {
			FollowerRoutine()

			// Exited, because need to start new elections cycle.
			TermNumber += 1

			// Election cycle. Convenient to put infinite cycle right here.
			for runElections() {
			}
		}
	}
}

func runElections() bool {
	grantsThreshold := QuorumSize - 1 // -1 for ourself
	// ClusterSize = 2k+1 => grantsThreshold = k ; k + 1 - majority
	// ClusterSize = 2k => grantsThreshold = k ; k + 1 - majority
	jitter := time.Duration(rand.Int63n(int64(time.Second)))
	timeout := RequestVoteTO + jitter

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for i := range ClusterSize {
		dstId := int64(i)
		if dstId == Idx {
			continue
		}

		go func(dstId int64) {
			response, _ := RequestVote(dstId, timeout)
			GrantsChannel <- response
		}(dstId)
	}

	// Try to collect votes.
	granted := int64(0)
outer:
	for range ClusterSize {
		select {
		case res := <-GrantsChannel:
			// If some host has newer Term, stop gathering votes.
			// Return false as sign that we should stop elections and continue `FollowerRoutine`.
			if res.Term > TermNumber {
				return false
			}
			if res.VoteGranted {
				granted += 1
			}
		case <-ctx.Done():
			// Just stop current elections cycle, but continue next time.
			break outer
		case <-AEChannel:
			// Some other host has been chosen as leader and started sending heartbits.
			// Return false as sign that we should stop elections and continue `FollowerRoutine`.
			return false
		}

		if granted >= grantsThreshold {
			break
		}
	}

	// If received enough grants, we set `IsLeader` to true (so LeaderRoutine will start).
	// Return false because we need to stop elections.
	if granted >= grantsThreshold {
		IsLeader = true
		return false
	}

	// Continue elections.
	return true
}

func handleAppendEntries(status string) {
	log.Printf("[INFO] [%s] received `append entries` request\n", status)
}

func handleRequestVote(status string) {
	log.Printf("[INFO] [%s] received `request vote` request\n", status)
}

func handleClientRequest(_ FromClientLogEntry, status string) {
	log.Printf("[INFO] [%s] received client api request\n", status)
}

func FollowerRoutine() {
	jitter := time.Duration(rand.Int63n(int64(time.Second)))

	timeToTick := jitter + IdleTO
	ticker := time.NewTicker(timeToTick)

	for {
		timeToTick = IdleTO + time.Duration(rand.Int63n(int64(time.Second)))
		select {
		case <-ticker.C:
			log.Println("[INFO] [follower] tick expired")
			log.Println("[INFO] [follower] beginning elections")
			return
		case <-AEChannel:
			handleAppendEntries("follower")
			ticker.Reset(timeToTick)
		case <-RVChannel:
			handleRequestVote("follower")
		case entry := <-ClientAPIChannel:
			handleClientRequest(entry, "follower")
		}
	}
}

// broadcastHeartbeat is used by LeaderRoutine to ping hosts.
func broadcastHeartbeat() {
	for i := range ClusterSize {
		if int64(i) == Idx {
			continue
		}

		entries := make([]LogEntry, 0)
		_, err := AppendEntries(int64(i), entries, 0, 0 /*zeros because fast path*/, 50*time.Millisecond)
		if err != nil {
			log.Printf("[INFO] [broadcastHeartbeat] i: %d ; error: %v\n", i, err)
		}
	}
}

func LeaderRoutine() {
	ticker := time.NewTicker(HeartbitTO)

	for {
		select {
		case t := <-ticker.C:
			log.Println("[INFO] [master] tick at", t)
			broadcastHeartbeat()

		// Here we handle that there are pings from other hosts with bigger term number.
		// If smth received from these channels, then `TermNumber` of sender is bigger than ours.
		case <-AEChannel:
			handleAppendEntries("master")
			IsLeader = false
			return
		case <-RVChannel:
			handleRequestVote("master")
			IsLeader = false
			return
		case entry := <-ClientAPIChannel:
			handleClientRequest(entry, "master")
		}
	}
}
