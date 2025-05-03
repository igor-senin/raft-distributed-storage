package cli

import (
	"errors"
	"flag"
	"fmt"
	"net"
	"net/netip"
	"strconv"
	"strings"

	"github.com/igor-senin/raft-distributed-storage/raft"
	"github.com/igor-senin/raft-distributed-storage/storage"
)

// InitCommandLine initializes some global variables
// using command line flags.
func InitCommandLine() error {
	flag.IntVar(&raft.ClusterSize, "clusterSize", 1, "total amount of replicas in raft group")
	flag.IntVar(&raft.Idx, "idx", 0, "index of this replica in range [0, clusterSize)")
	var baseAddr string
	flag.StringVar(&baseAddr, "baseAddr", "0.0.0.0", "base IP address for all replicas")

	flag.Parse()

	var err error = nil

	if baseAddr == "0.0.0.0" {
		err = errors.New("cli: you must explicitly specify IP addresses")
		goto out
	}

	storage.BaseIPAddr, err = netip.ParseAddr(baseAddr)
	if err != nil {
		goto out
	}

	if raft.Idx == 0 {
		raft.IsLeader = true
	}

	fmt.Printf("[Init CLI] ClusterSize = %d\n", raft.ClusterSize)
	fmt.Printf("[Init CLI] Idx = %d\n", raft.Idx)
	fmt.Printf("[Init CLI] BaseAddress = %s\n", baseAddr)

out:
	return err
}

// addToIp takes an IPv4 address as a string and an integer n,
// adds n to the address' last octet, and returns resulting IP as a string.
// It correctly handles overflow inside last octet.
func addToIP(ipStr string, n uint8) (string, error) {
	if net.ParseIP(ipStr).To4() == nil {
		return "", fmt.Errorf("invalid IPv4 address: %s", ipStr)
	}

	octets := strings.Split(ipStr, ".")

	m, err := strconv.ParseInt(octets[3], 0, 8)
	if err != nil {
		return "", err
	}
	octets[3] = string(uint8(m) + n)

	return strings.Join(octets, "."), nil
}
