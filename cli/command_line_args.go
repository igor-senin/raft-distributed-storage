package cli

import "flag"

var IsLeader bool

func InitCommandLine() {
	isLeaderPtr := flag.Bool("leader", false, "this instance is leader")
	flag.Parse()
	IsLeader = *isLeaderPtr
}
