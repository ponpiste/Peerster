// ========== CS-438 HW0 Skeleton ===========
// *** Implement here the handler for simple message processing ***

package gossip

import (
	"net"
	"encoding/json"
)

// Exec is the function that the gossiper uses to execute the handler for a SimpleMessage
// processSimple processes a SimpleMessage as such:
// - add message's relay address to the known peers
// - update the relay field
func (msg *SimpleMessage) Exec(g *Gossiper, addr *net.UDPAddr) error {

	var new_msg = SimpleMessage {
		OriginPeerName: msg.OriginPeerName,
		RelayPeerAddr: g.address,
		Contents: msg.Contents,
	}

	g.callback(msg.OriginPeerName, GossipPacket{&new_msg})

	b, err := json.Marshal(new_msg)
	if err != nil {
		return err
	}

	g.broadcast(b)
	g.AddAddresses(msg.RelayPeerAddr)

	return nil
}
