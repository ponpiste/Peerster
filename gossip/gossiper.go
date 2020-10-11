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
	"math/rand"
    "time"
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

	// watchers
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
	messages map[string][]string
	mongering map[string]*RumorMessage

	stop_chan chan int
	peers_mux sync.Mutex

	antiEntropy int
	routeTimer int

	// each gossiper
	// has its own random
	// generator. This
	// allows no avoid
	// seeding in main()
	source rand.Source
	ran *rand.Rand
}

// NewGossiper returns a Gossiper that is able to listen to the given address
// and which has the given identifier. The address must be a valid IPv4 UDP
// address. This method can panic if it is not possible to create a
// listener on that address. To run the gossip protocol, call `Run` on the
// gossiper.
func NewGossiper(address, identifier string, antiEntropy int, routeTimer int) (BaseGossiper, error) {

	g := Gossiper{

		inWatcher: watcher.NewSimpleWatcher(),
		outWatcher: watcher.NewSimpleWatcher(),

		Handlers: make(map[reflect.Type]interface{}),
		messages: make(map[string][]string),
		mongering: make(map[string]*RumorMessage),
		addr: address,
		identifier: identifier,
		peers: make([]*net.UDPAddr, 0),
		stop_chan: make(chan int),

		antiEntropy: antiEntropy,
		routeTimer: routeTimer,
	}

	g.source = rand.NewSource(time.Now().UnixNano())
	g.ran = rand.New(g.source)

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

	go g.runAntiEntropy()

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

		g.inWatcher.Notify(CallbackPacket{Addr: sender.String(), Msg: packet})

		if (packet.Simple != nil) {
			err = g.ExecuteHandler(packet.Simple, sender)
		} else if (packet.Rumor != nil) {
			err = g.ExecuteHandler(packet.Rumor, sender)
		}else if(packet.Status != nil) {
			err = g.ExecuteHandler(packet.Status, sender)
		}else if(packet.Private != nil) {
			err = g.ExecuteHandler(packet.Private, sender)
		}else {
			log.Error("Error parsing message: all fields were nil")
			continue
		}

		// Might happen sometimes
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

func (g *Gossiper) addMessage(id string, msg string) {

	// Todo: should we lock here ?
	g.messages[id] = append(g.messages[id], msg)
}

func (g *Gossiper) runAntiEntropy() {

	// Todo: how to stop this on stopMessage ?

	for {

		// Todo factor this in a function
		time.Sleep(time.Duration(g.antiEntropy) * time.Second)
		addr := g.randomPeer("")
		want := make([]PeerStatus, 0)
		g.map2slice(want)
		
		packet := GossipPacket {
			Status: &StatusPacket {
				Want: want,
			},
		}

		go g.send(packet, addr)
	}
}

func (g *Gossiper) getLatest(id string) uint32 {

	// Todo: should we lock here ?

	if val, ok := g.messages[id]; ok {
		return uint32(len(val))
	}else {
		return 0;
	}
}

func (g* Gossiper) map2slice(want []PeerStatus) {

	// Todo: lock a mutex here ?
	for key, value := range g.messages {

		want = append(want, PeerStatus {Identifier: key, NextID: uint32(1 + len(value))})
	}
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

// returns random peer, or nil of no was found
func (g *Gossiper) randomPeer(blacklisted string) *net.UDPAddr {

	g.peers_mux.Lock()
	defer g.peers_mux.Unlock()

	n := len(g.peers)
	if n == 0 {return nil}

	r := g.ran.Intn(n)
	if g.peers[r].String() != blacklisted {
		return g.peers[r]
	}

	s := (r+1)%n
	if s == r {return nil}

	// the addAddress function
	// guarantees that all addresses
	// are different
	return g.peers[s]

}

func (g *Gossiper) send(p GossipPacket, to *net.UDPAddr) {

	b, err := json.Marshal(p)

	// Should really never happen
	if err != nil {
		panic(fmt.Sprintf("Could not parse JSON: %v", err))
	}

	g.outWatcher.Notify(CallbackPacket{Addr: to.String(), Msg: p})

	_, err = g.conn.WriteToUDP(b, to)

	// Might happen once a day
	// The peer may have closed the socket
	if err != nil {
		log.Error("Could not write to UDP addr:", err)
	}
}

func (g *Gossiper) broadcast(p GossipPacket, blacklisted string) {

	g.peers_mux.Lock()
	defer g.peers_mux.Unlock()

	for _, peer := range g.peers {

		if peer.String() == blacklisted {continue}
		g.send(p, peer)
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

	go g.broadcast(packet, "")
}

// AddPrivateMessage sends the message to the next hop.
func (g *Gossiper) AddPrivateMessage(text, dest, origin string, hoplimit int) {
	log.Error("Implement me")
}

func (g *Gossiper) addAddress(addr *net.UDPAddr) {

	// Should really never happen
	if addr == nil {
		panic(fmt.Sprintf("Cannot add nil address"))
	}

	g.peers_mux.Lock()
	defer g.peers_mux.Unlock()

	for _, peer := range g.peers {

		if peer.String() == addr.String() {
			return
		}
	}

	g.peers = append(g.peers, addr)
}

// AddAddresses implements gossip.BaseGossiper. It takes any number of node
// addresses that the gossiper can contact in the gossiping network.
func (g *Gossiper) AddAddresses(addresses ...string) error {

	var err error = nil

	for _, addr := range addresses {

		udpAddr, err := net.ResolveUDPAddr("udp", addr)

		// Might happen sometimes
		if err != nil {
			err = xerrors.Errorf("At least one address couldn't be resolved")

			// log because in Exec() we use
			// AddAddress in a Go routine and
			// do not use the error
			log.Error("Error resolving address:", err)
			continue
		}

		g.addAddress(udpAddr)
	}
	return err
}

func (g* Gossiper) AddMessage(text string) uint32 {

	return 0
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

func (g *Gossiper) BroadcastMessage(GossipPacket) {

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
