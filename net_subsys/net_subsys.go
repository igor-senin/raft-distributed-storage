package net_subsys

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var BaseIPAddr string

func InitNet() error {
	log.Println("[Init Net]")
	return nil
}

// SendToId serves as main function to send single message to host with id dstId.
// Args:
//  1. dstId	: id of destination node
//  2. path 	: path part of url, i.e. rpc/append_entries, rpc/request_vote, etc
//  3. payload: json'able payload
//  4. timeout: if request is too long, stop it and return
func SendToId(dstId int64, path string, payload any, timeout time.Duration) (*http.Response, error) {
	dstIP, err := AddToIP(BaseIPAddr, dstId)
	if err != nil {
		return nil, err
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("http://%s:8786/%s", dstIP, path)

	client := http.Client{
		Timeout: timeout,
	}
	resp, err := client.Post(url, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// AddToIp takes an IPv4 address as a string and an integer n,
// adds n to the address' last octet, and returns resulting IP as a string.
// It correctly handles overflow inside last octet.
func AddToIP(ipStr string, n int64) (string, error) {
	if net.ParseIP(ipStr).To4() == nil {
		return "", fmt.Errorf("invalid IPv4 address: %s", ipStr)
	}

	octets := strings.Split(ipStr, ".")

	m, err := strconv.ParseInt(octets[3], 0, 8)
	if err != nil {
		return "", err
	}
	octets[3] = strconv.Itoa(int(uint8(m) + uint8(n)))

	return strings.Join(octets, "."), nil
}
