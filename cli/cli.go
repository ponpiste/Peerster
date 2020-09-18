// ========== CS-438 HW0 Skeleton ===========
// *** Implement here the CLI client ***

package main

import (
	"flag"
	"fmt"

	"go.dedis.ch/cs438/peerster/hw0/client"

	"go.dedis.ch/onet/v3/log"
)

func main() {
	UIPort := flag.String("UIPort", client.DefaultUIPort, "port for  gossip communication with peers")
	msg := flag.String("msg", "i just came to say hello", "message to be sent")
	flag.Parse()

	UIAddr := "http://127.0.0.1:" + *UIPort
	fmt.Println("client contacts", UIAddr, "with msg", *msg)

	sendMsg(UIAddr, &client.ClientMessage{Contents: *msg})

}

// sendMsg protobuf encodes the packet and sends it as an UDP datagram
// to the given address.
func sendMsg(address string, p *client.ClientMessage) {
	log.Error("Implment me")
}
