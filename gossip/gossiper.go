// ========== CS-438 HW0 Skeleton ===========
// *** Implement here the gossiper ***

package gossip

import (
	"context"
	"go.dedis.ch/cs438/hw1/gossip/watcher"
	"reflect"

	"go.dedis.ch/onet/v3/log"
)

// BaseGossipFactory provides a factory to instantiate a Gossiper
//
// - implements gossip.GossipFactory
type BaseGossipFactory struct{}

// stopMsg is used to notifier the listener when we want to close the
// connection, so that the listener knows it can stop listening.
const stopMsg = "stop"

// New implements gossip.GossipFactory. It creates a new gossiper.
func (f BaseGossipFactory) New(address, identifier string, antiEntropy int, routeTimer int) (BaseGossiper, error) {
	return NewGossiper(address, identifier, antiEntropy, routeTimer)
}

// Gossiper privides the functionalities to handle a distributes gossip
// protocol.
//
// - implements gossip.BaseGossiper
type Gossiper struct {
	inWatcher  watcher.Watcher
	outWatcher watcher.Watcher
	// routes holds the routes to different destinations. The key is the Origin,
	// or destination.
	routes map[string]*RouteStruct
	Handlers map[reflect.Type]interface{}
	// *** Add more here
}

// NewGossiper returns a Gossiper that is able to listen to the given address
// and which has the given identifier. The address must be a valid IPv4 UDP
// address. This method can panic if it is not possible to create a
// listener on that address. To run the gossip protocol, call `Run` on the
// gossiper.
func NewGossiper(address, identifier string, antiEntropy int, routeTimer int) (BaseGossiper, error) {
	log.Error("Implement me")

	// TODO define g with the following fields for the Watcher
	/*
	inWatcher: watcher.NewSimpleWatcher(),
	outWatcher: watcher.NewSimpleWatcher(),
	 */

	return nil, nil
}

// Run implements gossip.BaseGossiper. It starts the listening of UDP datagrams
// on the given address and starts the antientropy. This is a blocking function.
func (g *Gossiper) Run(ready chan struct{}) {
	log.Error("Implement me")
}

// Stop implements gossip.BaseGossiper. It closes the UDP connection
func (g *Gossiper) Stop() {
	log.Error("Implement me")
}

// AddSimpleMessage implements gossip.BaseGossiper. It takes a text that will be
// spread through the gossip network with the identifier of g.
func (g *Gossiper) AddSimpleMessage(text string) {
	log.Error("Implement me")
}

// AddPrivateMessage sends the message to the next hop.
func (g *Gossiper) AddPrivateMessage(text, dest, origin string, hoplimit int) {
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

// GetDirectNodes implements gossip.BaseGossiper. It returns the list of nodes whose routes are known to this node
func (g *Gossiper) GetDirectNodes() []string {
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

// GetRoutingTable implements gossip.BaseGossiper. It returns the known routes.
func (g *Gossiper) GetRoutingTable() map[string]*RouteStruct {
	log.Error("Implement me")
	return nil
}

// RegisterCallback implements gossip.BaseGossiper. It sets the callback that
// must be called each time a new message arrives.
func (g *Gossiper) RegisterCallback(m NewMessageCallback) {
	log.Error("Implement me")
}

// Watch implements gossip.BaseGossiper. It returns a chan populated with new
// incoming packets
func (g *Gossiper) Watch(ctx context.Context, fromIncoming bool) <-chan CallbackPacket {
	w := g.inWatcher

	if !fromIncoming {
		w = g.outWatcher
	}

	o := observer{make(chan CallbackPacket, 1)}

	w.Add(o)

	go func() {
		<-ctx.Done()
		w.Remove(o)
		close(o.c)
	}()

	return o.c
}

// - implements watcher.observable
type observer struct {
	c chan CallbackPacket
}

func (o observer) Notify(i interface{}) {
	o.c <- i.(CallbackPacket)
}

// An example of how to send an incoming packet to the Watcher
// g.inWatcher.Notify(CallbackPacket{Addr: addrStr, Msg: gossipPacket})