// ========== CS-438 HW0 Skeleton ===========
// *** Implement here the gossiper ***

package gossip

import (
	"go.dedis.ch/onet/v3/log"
	"reflect"
	"golang.org/x/xerrors"
)

// BaseGossipFactory provides a factory to instantiate a Gossiper
//
// - implements gossip.GossipFactory
type BaseGossipFactory struct{}

// stopMsg is used to notify the listener when we want to close the
// connection, so that the listener knows it can stop listening.
const stopMsg = "stop"

// New implements gossip.GossipFactory. It creates a new gossiper.
func (f BaseGossipFactory) New(address, identifier string) (BaseGossiper, error) {
	return NewGossiper(address, identifier)
}

// Gossiper provides the functionalities to handle a distributed gossip
// protocol.
//
// - implements gossip.BaseGossiper
type Gossiper struct {
	Handlers map[reflect.Type]interface{}
	// *** Add more here
}

// NewGossiper returns a Gossiper that is able to listen to the given address
// and which has the given identifier. The address must be a valid IPv4 UDP
// address. To run the gossip protocol and create the UDP connection, call `Run`
// on the gossiper.
func NewGossiper(address, identifier string) (BaseGossiper, error) {
	log.Error("Implement me")

	// TODO define a gossiper g, then enable the following code
	/*
	err := g.RegisterHandler(&SimpleMessage{})
	if err != nil {
		log.Fatal("failed to register", err)
	}
	*/
	return nil, xerrors.New("implement me")
}

// Run implements gossip.BaseGossiper. It creates the UDP connection and starts
// the listening of UDP datagrams on the given address, until `Stop` is called.
// This method can panic if it is not possible to create the connection on that
// address. The ready chan must be closed when it is running.
func (g *Gossiper) Run(ready chan struct{}) {
	log.Error("Implement me")
}

// Stop implements gossip.BaseGossiper. It closes the UDP connection. You should
// be sure that you stopped listening before closing the connection. For that
// you can send a stop message to the listener, which then knows that it can
// stop listening.
func (g *Gossiper) Stop() {
	log.Error("Implement me")
}

// AddSimpleMessage implements gossip.BaseGossiper. It takes a text that will be
// spread through the gossip network with the identifier of g.
func (g *Gossiper) AddSimpleMessage(text string) {
	log.Error("Implement me")
}

// AddAddresses implements gossip.BaseGossiper. It takes any number of node
// addresses that the gossiper can contact in the gossiping network.
func (g *Gossiper) AddAddresses(addresses ...string) error {
	log.Error("Implement me")
	return nil
}

// GetNodes implements gossip.BaseGossiper. It returns the list of nodes this
// gossiper knows currently in the network.
func (g *Gossiper) GetNodes() []string {
	log.Error("Implement me")
	return make([]string,0)
}

// SetIdentifier implements gossip.BaseGossiper. It changes the identifier sent
// with messages originating from this gossiper.
func (g *Gossiper) SetIdentifier(id string) {
	log.Error("Implement me")
}

// GetIdentifier implements gossip.BaseGossiper. It returns the currently used
// identifier for outgoing messages from this gossiper.
func (g *Gossiper) GetIdentifier() string {
	log.Error("Implement me")
	return ""
}

// RegisterCallback implements gossip.BaseGossiper. It sets the callback that
// must be called each time a new message arrives.
func (g *Gossiper) RegisterCallback(m NewMessageCallback) {
	log.Error("Implement me")
}
