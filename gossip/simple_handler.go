// ========== CS-438 HW0 Skeleton ===========
// *** Implement here the handler for simple message processing ***

package gossip

import (
	"net"
	"fmt"
	"encoding/json"
	"golang.org/x/xerrors"
)

// Exec is the function that the gossiper uses to execute the handler for a SimpleMessage
// processSimple processes a SimpleMessage as such:
// - add message's relay address to the known peers
// - update the relay field
func (msg *SimpleMessage) Exec(g *Gossiper, addr *net.UDPAddr) error {

	var new_msg = SimpleMessage {
		OriginPeerName: msg.OriginPeerName,
		RelayPeerAddr: g.addr,
		Contents: msg.Contents,
	}

	// the callback might block or be very long
	go g.callback(msg.OriginPeerName, GossipPacket{&new_msg})

	b, err := json.Marshal(new_msg)

	// Should really never happen
	if err != nil {
		panic(fmt.Sprintf("Could not parse JSON: %v", err))
	}

	go g.broadcast(b, msg.RelayPeerAddr)
	err = g.AddAddresses(msg.RelayPeerAddr)

	// Might happen once a day
	// In theory, anything could be in the packet
	if err != nil {
		return xerrors.Errorf(fmt.Sprintf("Error adding address: %v", err))
	}
	
	return nil
}
