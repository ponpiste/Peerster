// ========== CS-438 HW1 Skeleton ===========
// *** Implement here the handler for simple message processing ***

package gossip

import (
	"net"

	"golang.org/x/xerrors"
)

// Exec is the function that the gossiper uses to execute the handler for a SimpleMessage
// processSimple processes a SimpleMessage as such:
// - add message's relay address to the known peers
// - update the relay field
func (msg *SimpleMessage) Exec(g *Gossiper, addr *net.UDPAddr) error {
	return xerrors.Errorf("Implement me")
}

// Exec is the function that the gossiper uses to execute the handler for a RumorMessage
func (msg *RumorMessage) Exec(g *Gossiper, addr *net.UDPAddr) error {
	return xerrors.Errorf("Implement me")
}

// Exec is the function that the gossiper uses to execute the handler for a StatusMessage
func (msg *StatusPacket) Exec(g *Gossiper, addr *net.UDPAddr) error {
	return xerrors.Errorf("Implement me")
}

// Exec is the function that the gossiper uses to execute the handler for a PrivateMessage
func (msg *PrivateMessage) Exec(g *Gossiper, addr *net.UDPAddr) error {
	return xerrors.Errorf("Implement me")
}
