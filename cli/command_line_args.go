package cli

import (
	"errors"
	"flag"
	"log"
	"net/netip"

	"github.com/igor-senin/raft-distributed-storage/net_subsys"
	"github.com/igor-senin/raft-distributed-storage/raft"
)

// InitCommandLine initializes some global variables
// using command line flags.
func InitCommandLine() error {
	log.Println("[Init CLI]")

	flag.Int64Var(&raft.ClusterSize, "clusterSize", 1, "total amount of replicas in raft group")
	flag.Int64Var(&raft.Idx, "idx", 0, "index of this replica in range [0, clusterSize)")
	flag.BoolVar(&raft.IsLeader, "leader", false, "true if replica starts as leader")
	flag.StringVar(&net_subsys.BaseIPAddr, "baseAddr", "0.0.0.0", "base IP address for all replicas")

	flag.Parse()

	var err error = nil

	if net_subsys.BaseIPAddr == "0.0.0.0" {
		log.Printf("[ERROR] [Init CLI] base ip address can't be 0.0.0.0")
		err = errors.New("cli: you must explicitly specify IP addresses")
		goto out
	}

	_, err = netip.ParseAddr(net_subsys.BaseIPAddr)
	if err != nil {
		log.Printf("[ERROR] [Init CLI] invalid base ip : %s\n", net_subsys.BaseIPAddr)
		goto out
	}

	log.Printf("[DEBUG] [Init CLI] ClusterSize = %d\n", raft.ClusterSize)
	log.Printf("[DEBUG] [Init CLI] Idx = %d\n", raft.Idx)
	log.Printf("[DEBUG] [Init CLI] BaseAddress = %s\n", net_subsys.BaseIPAddr)

out:
	return err
}
