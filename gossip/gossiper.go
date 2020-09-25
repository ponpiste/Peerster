// ========== CS-438 HW0 Skeleton ===========
// *** Implement here the gossiper ***

package gossip

import (
	"go.dedis.ch/onet/v3/log"
	"reflect"
	"net"
	"fmt"
	"sync"
	"encoding/json"
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
	addr string
	identifier string
	conn *net.UDPConn
	udpAddr *net.UDPAddr
	callback NewMessageCallback
	peers []*net.UDPAddr

	stop_chan chan int
	peers_mux sync.Mutex
}

// NewGossiper returns a Gossiper that is able to listen to the given address
// and which has the given identifier. The address must be a valid IPv4 UDP
// address. To run the gossip protocol and create the UDP connection, call `Run`
// on the gossiper.
func NewGossiper(address, identifier string) (BaseGossiper, error) {

	g := Gossiper{
		Handlers: make(map[reflect.Type]interface{}),
		addr: address,
		identifier: identifier,
		peers: make([]*net.UDPAddr, 0),
		stop_chan: make(chan int),
	}

	err := g.RegisterHandler(&SimpleMessage{})

	// Should really never happen
	if err != nil {
		panic(fmt.Sprintf("Could not register handler: %v", err))
	}
	
	return &g, nil
}

// Run implements gossip.BaseGossiper. It creates the UDP connection and starts
// the listening of UDP datagrams on the given address, until `Stop` is called.
// This method can panic if it is not possible to create the connection on that
// address. The ready chan must be closed when it is running.
func (g *Gossiper) Run(ready chan struct{}) {

	udpAddr, err := net.ResolveUDPAddr("udp", g.addr)

	// Should really never happen
	// We cannot continue if there is an error here
	if err != nil {
		panic(fmt.Sprintf("Could not resolve UDP addr: %v", err))
	}
	g.udpAddr = udpAddr
	g.conn, err = net.ListenUDP("udp", g.udpAddr)

	// Should really never happen
	if err != nil {
		panic(fmt.Sprintf("Could not listen to UDP addr: %v", err))
	}

	ready <- struct{}{}

	b := make([]byte, 1024)

	for  {

		n, sender, err := g.conn.ReadFromUDP(b)

		// Should really never happen
		if err != nil {
			panic(fmt.Sprintf("Could not read from UDP addr: %v", err))
		}

		if string(b[:n]) == stopMsg {
			g.conn.Close()
			g.stop_chan <- 1
			break
		}

		var msg SimpleMessage
		err = json.Unmarshal(b[:n], &msg)

		// Might happen once a day
		// In theory, we could receive anything
		if err != nil {
			log.Error("Error parsing message:", err)
		}

		// Might happen sometimes
		err = g.ExecuteHandler(&msg, sender)
		if err != nil {
			log.Error("Error executing handler:", err)
		}


		fmt.Printf("SIMPLE MESSAGE origin %v from %v contents %v\n", 
			msg.OriginPeerName, msg.RelayPeerAddr, msg.Contents)

		g.peers_mux.Lock()
		fmt.Printf("PEERS ")
		for i, peer := range g.peers {
			if i==0 {
				fmt.Printf("%v",peer)
			} else {
				fmt.Printf(",%v",peer)
			}
		}
		g.peers_mux.Unlock()
		fmt.Println()
	}
}

// Stop implements gossip.BaseGossiper. It closes the UDP connection. You should
// be sure that you stopped listening before closing the connection. For that
// you can send a stop message to the listener, which then knows that it can
// stop listening.
func (g *Gossiper) Stop() { 
	
	_, err := g.conn.WriteToUDP([]byte(stopMsg), g.udpAddr)

	// Should really never happen
	// We should be able to write to ourselves
	if err != nil {
		panic(fmt.Sprintf("Could not write to UDP addr: %v", err))
	}
	<- g.stop_chan
}

func (g *Gossiper) broadcast(b []byte, blacklisted string) {

	g.peers_mux.Lock()
	defer g.peers_mux.Unlock()

	for _, peer := range g.peers {

		if peer.String() == blacklisted {continue}

		_, err := g.conn.WriteToUDP(b, peer)

		// Might happen once a day
		// The peer may have closed the socket
		if err != nil {
			log.Error("Could not write to UDP addr:", err)
		}
	}
}

// AddSimpleMessage implements gossip.BaseGossiper. It takes a text that will be
// spread through the gossip network with the identifier of g.
func (g *Gossiper) AddSimpleMessage(text string) {

	fmt.Printf("CLIENT MESSAGE %v\n", text)
	g.peers_mux.Lock()
	fmt.Printf("PEERS ")
	for i, peer := range g.peers {
		if i==0 {
			fmt.Printf("%v",peer)
		} else {
			fmt.Printf(",%v",peer)
		}
	}
	g.peers_mux.Unlock()
	fmt.Println()

	var msg = SimpleMessage {
		OriginPeerName: g.identifier,
		RelayPeerAddr: g.addr,
		Contents: text,
	}

	b, err := json.Marshal(msg)

	// Should really never happen
	if err != nil {
		panic(fmt.Sprintf("Error marshalling the json: %v", err))
	}

	go g.broadcast(b, "")
}

// AddAddresses implements gossip.BaseGossiper. It takes any number of node
// addresses that the gossiper can contact in the gossiping network.
func (g *Gossiper) AddAddresses(addresses ...string) error {

	var err error = nil

	for _, addr := range addresses {

		udpAddr, err := net.ResolveUDPAddr("udp", addr)

		// Might happen sometimes
		if err != nil {
			err = xerrors.Errorf("At least one address couldn't be resolved",)
			log.Error("Error resolving address:", err)
		}

		g.peers_mux.Lock()

		found := false
		for _, peer := range g.peers {
			if peer.String() == udpAddr.String() {
				found = true
				break
			}
		}

		if !found {
			g.peers = append(g.peers, udpAddr)
		}
		g.peers_mux.Unlock()
	}
	return err
}

// GetNodes implements gossip.BaseGossiper. It returns the list of nodes this
// gossiper knows currently in the network.
func (g *Gossiper) GetNodes() []string {

	g.peers_mux.Lock()
	defer g.peers_mux.Unlock()

	// return a copy otherwise the caller
	// gets a pointer to the real peers which
	// can change and which he can modify

	cpy := make([]string, len(g.peers))
	for i, peer := range g.peers {
		cpy[i] = peer.String()
	}
	return cpy
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
