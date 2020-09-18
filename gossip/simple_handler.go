// ========== CS-438 HW0 Skeleton ===========
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