// ========== CS-438 HW0 Skeleton ===========
// *** Implement here the gossiper ***

package gossip

import (
	"context"
	"go.dedis.ch/cs438/hw1/gossip/watcher"
	"reflect"
	"net"
	"fmt"
	"sync"
	"encoding/json"
	"golang.org/x/xerrors"

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
// address. This method can panic if it is not possible to create a
// listener on that address. To run the gossip protocol, call `Run` on the
// gossiper.
func NewGossiper(address, identifier string, antiEntropy int, routeTimer int) (BaseGossiper, error) {

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

// Run implements gossip.BaseGossiper. It starts the listening of UDP datagrams
// on the given address and starts the antientropy. This is a blocking function.
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

	// The usual size of a MTU
	// is 1500 bytes
	b := make([]byte, 1500)

	for  {

		n, sender, err := g.conn.ReadFromUDP(b)

		// Should really never happen
		if err != nil {
			panic(fmt.Sprintf("Could not read from UDP addr: %v", err))
		}

		// Close after having received the packet
		// otherwise the connection is closed
		// while reading
		if string(b[:n]) == stopMsg {
			g.conn.Close()
			g.stop_chan <- 1
			break
		}

		var packet GossipPacket
		err = json.Unmarshal(b[:n], &packet)

		// Might happen once a day
		// In theory, we could receive anything
		if err != nil {
			log.Error("Error parsing message:", err)
			continue
		}

		// Might happen sometimes
		err = g.ExecuteHandler(packet.Simple, sender)
		if err != nil {
			log.Error("Error executing handler:", err)
			continue
		}


		fmt.Printf("SIMPLE MESSAGE origin %v from %v contents %v\n", 
			packet.Simple.OriginPeerName, packet.Simple.RelayPeerAddr, packet.Simple.Contents)
		g.printPeers()
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

	// Stop() blocks until the connection
	// is closed. Avoids same port being reused
	// after a call to Stop() and which is 
	// not closed yet
	<- g.stop_chan
}

func (g *Gossiper) printPeers() {

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
	g.printPeers()

	var msg = SimpleMessage {
		OriginPeerName: g.identifier,
		RelayPeerAddr: g.addr,
		Contents: text,
	}

	var packet = GossipPacket {
		Simple: &msg,
	}

	b, err := json.Marshal(packet)

	// Should really never happen
	if err != nil {
		panic(fmt.Sprintf("Error marshalling the json: %v", err))
	}

	go g.broadcast(b, "")
}

// AddPrivateMessage sends the message to the next hop.
func (g *Gossiper) AddPrivateMessage(text, dest, origin string, hoplimit int) {
	log.Error("Implement me")
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

			// log because in Exec() we use
			// AddAddress in a Go routine and
			// do not use the error
			log.Error("Error resolving address:", err)
			continue
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

// GetDirectNodes implements gossip.BaseGossiper. It returns the list of nodes whose routes are known to this node
func (g *Gossiper) GetDirectNodes() []string {
	log.Error("Implement me")
	return make([]string,0)
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

// GetRoutingTable implements gossip.BaseGossiper. It returns the known routes.
func (g *Gossiper) GetRoutingTable() map[string]*RouteStruct {
	log.Error("Implement me")
	return nil
}

// RegisterCallback implements gossip.BaseGossiper. It sets the callback that
// must be called each time a new message arrives.
func (g *Gossiper) RegisterCallback(m NewMessageCallback) {
	g.callback = m
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
