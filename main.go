// ========== CS-438 HW1 Skeleton ===========
// *** Do not change this file ***

// This file should be the entering point to your program.
// Here, we only parse the input and start the logic implemented
// in other files.
package main

import (
	"flag"
	"strings"

	"go.dedis.ch/cs438/hw1/gossip"
	"go.dedis.ch/cs438/hw1/client"

	"go.dedis.ch/onet/v3/log"
)

const defaultGossipAddr = "127.0.0.1:33000" // IP address:port number for gossiping
const defaultName = "peerXYZ"               // Give a unique default name

func main() { 

	UIPort := flag.String("UIPort", client.DefaultUIPort, "port for gossip communication with peers")
	antiEntropy := flag.Int("antiEntropy", 10, "timeout in seconds for anti-entropy (relevant only fo rPart2)' default value 10 seconds.")
	gossipAddr := flag.String("gossipAddr", defaultGossipAddr, "ip:port for gossip communication with peers")
	ownName := flag.String("name", defaultName, "identifier used in the chat")
	peers := flag.String("peers", "", "peer addresses used for bootstrap")
	broadcastMode := flag.Bool("broadcast", true, "run gossiper in broadcast mode")
	routeTimer := flag.Int("rtimer", 0, "route rumors sending period in seconds, 0 to disable sending of route rumors (default)")
	flag.Parse()

	UIAddress := "127.0.0.1:" + *UIPort
	gossipAddress := *gossipAddr
	bootstrapAddr := strings.Split(*peers, ",")

	// The work happens in the gossip folder. You should not touch the code in
	// this package.
	fac := gossip.GetFactory()
	g, err := fac.New(gossipAddress, *ownName, *antiEntropy, *routeTimer)
	if err != nil {
		panic(err)
	}

	if bootstrapAddr[0] != "" {
		g.AddAddresses(bootstrapAddr...)
	}

	controller := NewController(*ownName, UIAddress, gossipAddress, *broadcastMode, g, bootstrapAddr...)

	ready := make(chan struct{})
	go g.Run(ready)
	defer g.Stop()
	<-ready
	controller.Run()

	log.SetDebugVisible(1)
}
