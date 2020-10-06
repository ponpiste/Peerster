// ========== CS-438 HW1 Skeleton ===========
// *** Implement here the CLI client ***

package main

import (
	"bytes"
	"flag"
	"fmt"
	"net/http"
	"encoding/json"
	"go.dedis.ch/cs438/peerster/hw0/client"

	"go.dedis.ch/cs438/hw1/client"

	"go.dedis.ch/onet/v3/log"
)

func main() {
	UIPort := flag.String("UIPort", client.DefaultUIPort, "port for  gossip communication with peers")
	msg := flag.String("msg", "i just came to say hello", "message to be sent")
	dest := flag.String("dest", "", "destination for the private message")
	flag.Parse()

	UIAddr := "http://127.0.0.1:" + *UIPort
	fmt.Println("client contacts", UIAddr, "with msg", *msg)

	if dest != nil {
		fmt.Println("Destination is:", *dest)
	}

	if *msg != "" {
		println("Sending private message or normal depending on whether Destination is present or not respectively")
		sendMsg(UIAddr, &client.ClientMessage{Contents: *msg, Destination: *dest})
		return
	}

}

// sendMsg json encodes the packet and sends it as an UDP datagram
// to the given address + "/message"
// Note that it must be able to handle ClientMessage.Destination now
func sendMsg(address string, p *client.ClientMessage) {

	b, err := json.Marshal(p)

	// Should really never happen
	if err != nil {
		panic(fmt.Sprintf("Failed to marshal client message: %v", err))
	}

	_, err = http.Post(address + "/message", "application/json", bytes.NewBuffer(b))

	// Might happen once a day
	if err != nil {
		log.Error("failed to send http post", err)
	    return
	}
}
