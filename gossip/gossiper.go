// ========== CS-438 HW0 Skeleton ===========
// *** Implement here the gossiper ***

package gossip

import (
	"go.dedis.ch/onet/v3/log"
	"reflect"
	"net"
	"fmt"
	"encoding/json"
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
	address string
	identifier string
	conn *net.UDPConn
	udpAddr *net.UDPAddr
	callback NewMessageCallback
	peers []string
}

// NewGossiper returns a Gossiper that is able to listen to the given address
// and which has the given identifier. The address must be a valid IPv4 UDP
// address. To run the gossip protocol and create the UDP connection, call `Run`
// on the gossiper.
func NewGossiper(address, identifier string) (BaseGossiper, error) {

	g := Gossiper{
		Handlers: make(map[reflect.Type]interface{}),
		address: address,
		identifier: identifier,
		peers: make([]string, 0),
	}

	err := g.RegisterHandler(&SimpleMessage{})
	if err != nil {
		log.Fatal("failed to register", err)
	}
	
	return &g, nil
}

// Run implements gossip.BaseGossiper. It creates the UDP connection and starts
// the listening of UDP datagrams on the given address, until `Stop` is called.
// This method can panic if it is not possible to create the connection on that
// address. The ready chan must be closed when it is running.
func (g *Gossiper) Run(ready chan struct{}) {

	var err error
	g.udpAddr, err = net.ResolveUDPAddr("udp4", g.address)
	if err != nil {
		panic(fmt.Sprintf("Could not resolve UDP addr: %v", err))
	}

	g.conn, err = net.ListenUDP("udp4", g.udpAddr)
	if err != nil {
		panic(fmt.Sprintf("Could not listen to UDP addr: %v", err))
	}

	ready <- struct{}{}

	b := make([]byte, 1024)

	for  {

		n, sender, err := g.conn.ReadFromUDP(b)
		if err != nil {
			panic(fmt.Sprintf("Could not read from UDP addr: %v", err))
		}

		fmt.Println()
		fmt.Println("Received: ")
		for _, x := range b[:n] {
			fmt.Printf(" %v",x)
		}

		if string(b[:n]) == stopMsg {
			g.conn.Close()
			break
		}

		var msg SimpleMessage
		err = json.Unmarshal(b[:n], &msg)

		if err != nil {
			panic(fmt.Sprintf("Error unmarshalling the json: %v", err))
		}

		g.ExecuteHandler(&msg, sender)
	}
}

// Stop implements gossip.BaseGossiper. It closes the UDP connection. You should
// be sure that you stopped listening before closing the connection. For that
// you can send a stop message to the listener, which then knows that it can
// stop listening.
func (g *Gossiper) Stop() {
	
	_, err := g.conn.WriteToUDP([]byte(stopMsg), g.udpAddr)
	if err != nil {
		panic(fmt.Sprintf("Could not write to UDP addr: %v", err))
	}
}

func (g *Gossiper) broadcast(b []byte) {

	fmt.Println()
	fmt.Println("Sent: ")
	for _, x := range b {
		fmt.Printf(" %v",x)
	}

	for _, peer := range g.peers {

		addr, err := net.ResolveUDPAddr("udp4", peer)
		if err != nil {
			panic(fmt.Sprintf("Could not resolve UDP addr: %v", err))
		}

		_, err = g.conn.WriteToUDP(b, addr)
		if err != nil {
			panic(fmt.Sprintf("Could not write to UDP addr: %v", err))
		}
	}
}

// AddSimpleMessage implements gossip.BaseGossiper. It takes a text that will be
// spread through the gossip network with the identifier of g.
func (g *Gossiper) AddSimpleMessage(text string) {

	var msg = SimpleMessage {
		OriginPeerName: g.identifier,
		RelayPeerAddr: g.address,
		Contents: text,
	}

	b, err := json.Marshal(msg)
	if err != nil {
		panic(fmt.Sprintf("Error marshalling the json: %v", err))
	}

	g.broadcast(b)
}

// AddAddresses implements gossip.BaseGossiper. It takes any number of node
// addresses that the gossiper can contact in the gossiping network.
func (g *Gossiper) AddAddresses(addresses ...string) error {
	
	g.peers = append(g.peers, addresses...)
	return nil
}

// GetNodes implements gossip.BaseGossiper. It returns the list of nodes this
// gossiper knows currently in the network.
func (g *Gossiper) GetNodes() []string {
	
	return g.peers
}

// SetIdentifier implements gossip.BaseGossiper. It changes the identifier sent
// with messages originating from this gossiper.
func (g *Gossiper) SetIdentifier(id string) {
	g.identifier = id
}

// GetIdentifier implements gossip.BaseGossiper. It returns the currently used
// identifier for outgoing messages from this gossiper.
func (g *Gossiper) GetIdentifier() string {
	return g.identifier
}

// RegisterCallback implements gossip.BaseGossiper. It sets the callback that
// must be called each time a new message arrives.
func (g *Gossiper) RegisterCallback(m NewMessageCallback) {
	g.callback = m
}
