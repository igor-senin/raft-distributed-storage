package main

import (
	"log"
	"net/http"

	"github.com/igor-senin/raft-distributed-storage/cli"
	"github.com/igor-senin/raft-distributed-storage/net_subsys"
	"github.com/igor-senin/raft-distributed-storage/raft"
	"github.com/igor-senin/raft-distributed-storage/storage"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	var err error = nil

	log.Println("startup")

	err = cli.InitCommandLine()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("cli init success")

	err = storage.InitStorage()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("storage init success")

	err = net_subsys.InitNet()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("net_subsys init success")

	go func() {
		http.HandleFunc("/", HandleRoot)
		http.HandleFunc("/resource", HandleResource)
		http.HandleFunc("/rpc/append_entries", HandleRPCAppendEntries)
		http.HandleFunc("/rpc/request_vote", HandleRPCRequestVote)

		addr := "0.0.0.0:8786"
		log.Println("listening on", addr)
		log.Fatal(http.ListenAndServe(addr, nil))
	}()

	raft.RunRaft()
}
