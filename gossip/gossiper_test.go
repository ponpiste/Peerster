package gossip

import (
	"context"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"math/rand"
	"net"
	"strconv"
	"sync"
	"testing"
	"time"
)

var factory = GetFactory()


func TestGossiper_Topo1_2Nodes_RumorReceived(t *testing.T) {
	antiEntropy := 1000
	routeTimer := 100
	n1, addr1 := createNode(t, "A", antiEntropy, routeTimer)
	n2, addr2 := createNode(t, "B", antiEntropy, routeTimer)
	addAddresses(t, n1, addr2)
	addAddresses(t, n2, addr1)

	ctx, cancel := context.WithTimeout(context.Background(),
		3*time.Second)
	defer cancel()

	out1 := n1.Watch( ctx, false)
	in2 := n2.Watch( ctx, true)

	startNodesBlocking(t, n1, n2)
	defer n1.Stop()
	defer n2.Stop()

	const expectedMessage string = "From A to B, with love"

	finished := make(chan struct{})

	n2.RegisterCallback(
		func(origin string, message GossipPacket) {
			require.Equal(t, n1.GetIdentifier(), origin)
			require.Equal(t, n1.GetIdentifier(), message.Rumor.Origin)
			require.Contains(t, expectedMessage, message.Rumor.Text)
			close(finished)
		})

	n1.AddMessage(expectedMessage)

	out := waitRumorMsg(t, ctx, out1)
	require.Equal(t, addr2, out.Addr)
	require.NotNil(t, out.Msg.Rumor)
	require.Equal(t, expectedMessage, out.Msg.Rumor.Text)

	in := waitRumorMsg(t, ctx, in2)
	require.Equal(t, addr1, in.Addr)
	require.NotNil(t, in.Msg.Rumor)
	require.Equal(t, expectedMessage, in.Msg.Rumor.Text)

	waitChannel(t, ctx, finished)
}

func TestGossiper_Topo1_2Nodes_AckRumor(t *testing.T) {
	antiEntropy := 1000
	routeTimer := 100
	n1, addr1 := createNode(t, "A", antiEntropy, routeTimer)
	n2, addr2 := createNode(t, "B", antiEntropy, routeTimer)
	addAddresses(t, n1, addr2)
	addAddresses(t, n2, addr1)

	ctx, cancel := context.WithTimeout(context.Background(),
		3*time.Second)
	defer cancel()

	out1 := n1.Watch( ctx, false)
	out2 := n2.Watch( ctx, false)
	in1 := n1.Watch( ctx, true)

	msgRecN2 := streamIncomingGossips(n2)

	startNodesBlocking(t, n1, n2)
	defer n1.Stop()
	defer n2.Stop()

	const expectedMessage string = "From A to B, with love"

	n1.AddMessage(expectedMessage)

	rumorOut := waitRumorMsg(t, ctx, out1)
	require.Equal(t, addr2, rumorOut.Addr)
	require.NotNil(t, rumorOut.Msg.Rumor)
	require.Equal(t, expectedMessage, rumorOut.Msg.Rumor.Text)

	packet := <- msgRecN2
	require.NotNil(t, packet.Rumor, "Expected to receive a rumor message")
	require.Equal(t, n1.GetIdentifier(), packet.Rumor.Origin)
	require.Contains(t, expectedMessage, packet.Rumor.Text)

	expectedWant := []PeerStatus {
		{ Identifier: n1.GetIdentifier(), NextID: 2 },
	}

	statusOut := waitStatusMsg(t, ctx, out2)
	require.Equal(t, addr1, statusOut.Addr)
	require.NotNil(t, statusOut.Msg.Status)
	assert.ElementsMatch(t, expectedWant, statusOut.Msg.Status.Want)

	statusIn := waitStatusMsg(t, ctx, in1)
	require.NotNil(t, statusIn.Msg.Status,
		"Expected to receive a status message")
	assert.ElementsMatch(t, expectedWant, statusIn.Msg.Status.Want)
}

func TestGossiper_Topo1_2Nodes_AntiEntropy(t *testing.T) {
	antiEntropy := 1
	routeTimer := 100
	n1, addr1 := createNode(t, "A", antiEntropy, routeTimer)
	n2, addr2 := createNode(t, "B", antiEntropy, routeTimer)
	addAddresses(t, n1, addr2)
	addAddresses(t, n2, addr1)

	ctx, cancel := context.WithTimeout(context.Background(),
		5*time.Second)
	defer cancel()

	in1 := n2.Watch( ctx, true)

	startNodesBlocking(t, n1, n2)
	defer n1.Stop()
	defer n2.Stop()

	i := 0
	for {
		select {
		case p := <- in1:
			if p.Msg.Status != nil {
				i++
			}
		case <- ctx.Done():
			require.GreaterOrEqual(t, i, 4)
			require.LessOrEqual(t, i, 7)
			return
		}
	}
}

func TestGossiper_Topo2_3Nodes_AntiEntropy(t *testing.T) {
	// arrange
	antiEntropy := 1
	routeTimer := 100
	n1, addr1 := createNode(t, "A", antiEntropy, routeTimer)
	n2, addr2 := createNode(t, "B", antiEntropy, routeTimer)
	n3, addr3 := createNode(t, "C", antiEntropy, routeTimer)
	addAddresses(t, n1, addr2)
	addAddresses(t, n2, addr3)

	const expectedMessage string = "From B to C, with love"
	id2 := n2.GetIdentifier()

	msgRecN1 := streamIncomingGossips(n1)
	msgRecN3 := streamIncomingGossips(n3)

	// act
	startNodesBlocking(t, n1, n2, n3)
	defer n1.Stop()
	defer n2.Stop()
	defer n3.Stop()

	n2.AddMessage(expectedMessage)

	// assert
	msgN1Count := 0
	msgN3Count := 0
	for {
		select {
		case m := <- msgRecN1:
			require.Equal(t, id2, m.Rumor.Origin)
			require.Contains(t, expectedMessage, m.Rumor.Text)
			msgN1Count++
		case m := <- msgRecN3:
			require.Equal(t, id2, m.Rumor.Origin)
			require.Contains(t, expectedMessage, m.Rumor.Text)
			msgN3Count++
		case <- time.After(5*time.Second):
			require.Equal(t, 1, msgN1Count, "Expected A to receive 1 message")
			require.Equal(t, 1, msgN3Count, "Expected C to receive 1 message")
			require.Equal(t, 2, len(n2.GetNodes()))
			require.Contains(t, n2.GetNodes(), addr1)
			require.Contains(t, n2.GetNodes(), addr3)
			return
		}
	}
}

func TestGossiper_Topo3_3Nodes_AntiEntropy(t *testing.T) {
	// arrange
	antiEntropy12 := 1
	antiEntropy3 := 5
	routeTimer := 100
	n1, addr1 := createNode(t, "A", antiEntropy12, routeTimer)
	n2, addr2 := createNode(t, "B", antiEntropy12, routeTimer)
	n3, addr3 := createNode(t, "C", antiEntropy3, routeTimer)
	addAddresses(t, n1, addr2)
	addAddresses(t, n2, addr3)
	addAddresses(t, n3, addr1)

	const expectedMessage1 string = "I believe I can fly!"
	const expectedMessage2 string = "I believe I can touch the sky!"
	const msgCount int = 5
	id1 := n1.GetIdentifier()
	id2 := n2.GetIdentifier()

	msgRecN1 := streamIncomingGossips(n1)
	msgRecN2 := streamIncomingGossips(n2)

	// act
	startNodesBlocking(t, n1, n3)
	defer n1.Stop()
	defer n3.Stop()

	ctx, cancel := context.WithTimeout(context.Background(),
		10*time.Second)
	defer cancel()

	go func() {
		<- time.After(5*time.Second)
		startNodesBlocking(t, n2)
		defer n2.Stop()

		for i := 0; i<msgCount; i++ {
			n2.AddMessage(expectedMessage2)
		}

		<- ctx.Done()
	}()

	for i := 0; i<msgCount; i++ {
		n1.AddMessage(expectedMessage1)
	}

	// assert
	msgN1Count := 0
	msgN2Count := 0
	for {
		select {
		case m := <- msgRecN1:
			require.Equal(t, id2, m.Rumor.Origin)
			require.Contains(t, expectedMessage2, m.Rumor.Text)
			msgN1Count++
		case m := <- msgRecN2:
			require.Equal(t, id1, m.Rumor.Origin)
			require.Contains(t, expectedMessage1, m.Rumor.Text)
			msgN2Count++
		case <- ctx.Done():
			require.Equal(t, msgCount, msgN1Count)
			require.Equal(t, msgCount, msgN2Count)
			return
		}
	}
}

func TestGossiper_Topo4_5Nodes(t *testing.T) {
	// arrange
	antiEntropyAB := 1
	antiEntropyC := 1000
	routeTimer := 100
	n1, addr1 := createNode(t, "C1", antiEntropyC, routeTimer)
	n2, addr2 := createNode(t, "C2", antiEntropyC, routeTimer)
	n3, addr3 := createNode(t, "C3", antiEntropyC, routeTimer)
	n4, addr4 := createNode(t, "C4", antiEntropyC, routeTimer)
	n5, addr5 := createNode(t, "C5", antiEntropyC, routeTimer)
	n6, addr6 := createNode(t, "A", antiEntropyAB, routeTimer)
	n7, addr7 := createNode(t, "B", antiEntropyAB, routeTimer)

	cSet := make(map[string]BaseGossiper)
	cSet[addr1] = n1
	cSet[addr2] = n2
	cSet[addr3] = n3
	cSet[addr4] = n4
	cSet[addr5] = n5

	// link A <-- C1-5
	for _, n := range cSet {
		addAddresses(t, n, addr6)
	}

	// link A<->B
	addAddresses(t, n7, addr6)
	addAddresses(t, n6, addr7)

	// act
	startNodesBlocking(t, n1, n2, n3, n4, n5, n6, n7)
	defer func() {
		for _, n := range cSet {
			n.Stop()
		}
		defer n6.Stop()
		defer n7.Stop()
	}()

	ctx, cancel := context.WithTimeout(context.Background(),
		11*time.Second)
	defer cancel()
	out6 := n6.Watch( ctx, false)

	msg := "Hey there!"
	for _, n := range cSet {
		n.AddMessage(msg)
	}

	// assert
	statusPacketsToC := 0
	for {
		select {
		case p := <- out6:
			if p.Msg.Status != nil {
				if _, ok := cSet[p.Addr] ; ok {
					statusPacketsToC++
				}
			}
		case <- ctx.Done():
			for addr, _ := range cSet {
				require.Contains(t, n6.GetNodes(), addr)
			}
			require.GreaterOrEqual(t, statusPacketsToC, 1)
			return
		}
	}
}

func TestGossiper_Topo1_5Nodes_DSDV1(t *testing.T) {
	// arrange
	antiEntropy := 10
	routeTimer := 100
	numberOfNodes := 5
	nodeAddr := make(map[string]string)      // name -> addr
	nodeId := make(map[string]string)      // name -> identifier
	nodeSet := make(map[string]BaseGossiper) // name -> node
	nodeSlice := make([]BaseGossiper, numberOfNodes)

	// generate nodes
	for i := 0; i<numberOfNodes; i++ {
		id := string(byte('A')+byte(i)) // compute A-E
		n, addr := createNode(t, id, antiEntropy, routeTimer)
		nodeSet[id] = n
		nodeAddr[id] = addr
		nodeSlice[i] = n
		nodeId[id] = n.GetIdentifier()
	}

	n := func (name string) BaseGossiper {
		return nodeSet[name]
	}

	addAddresses(t, n("A"), nodeAddr["B"], nodeAddr["C"])
	addAddresses(t, n("B"), nodeAddr["D"], nodeAddr["E"])
	addAddresses(t, n("C"), nodeAddr["A"])
	addAddresses(t, n("D"), nodeAddr["B"])
	addAddresses(t, n("E"), nodeAddr["B"])

	// act
	startNodesBlocking(t, nodeSlice...)
	defer func() {
		for _, n := range nodeSlice {
			n.Stop()
		}
	}()

	for _, n := range nodeSet {
		n.AddMessage("I am alive!")
	}

	<- time.After(5*time.Second)

	// assert
	rtA := n("A").GetRoutingTable()
	require.Contains(t, rtA, nodeId["B"])
	require.Equal(t, nodeAddr["B"], rtA[nodeId["B"]].NextHop)
	require.Contains(t, rtA, nodeId["C"])
	require.Equal(t, nodeAddr["C"], rtA[nodeId["C"]].NextHop)

	rtD := n("D").GetRoutingTable()
	require.Contains(t, rtD, nodeId["B"])
	require.Equal(t, nodeAddr["B"], rtD[nodeId["B"]].NextHop)
}

func TestGossiper_Topo1_5Nodes_DSDV2(t *testing.T) {
	// arrange
	antiEntropy := 10
	routeTimer := 100
	numberOfNodes := 5
	nodeAddr := make(map[string]string)      // name -> addr
	nodeId := make(map[string]string)      // name -> identifier
	nodeSet := make(map[string]BaseGossiper) // name -> node
	nodeSlice := make([]BaseGossiper, numberOfNodes)

	// generate nodes
	for i := 0; i<numberOfNodes; i++ {
		id := string(byte('A')+byte(i)) // compute A-E
		n, addr := createNode(t, id, antiEntropy, routeTimer)
		nodeSet[id] = n
		nodeAddr[id] = addr
		nodeId[id] = n.GetIdentifier()
		nodeSlice[i] = n
	}

	n := func (name string) BaseGossiper {
		return nodeSet[name]
	}

	addAddresses(t, n("A"), nodeAddr["B"], nodeAddr["C"])
	addAddresses(t, n("B"), nodeAddr["D"], nodeAddr["E"])
	addAddresses(t, n("C"), nodeAddr["A"])
	addAddresses(t, n("D"), nodeAddr["B"])
	addAddresses(t, n("E"), nodeAddr["B"])

	// act
	startNodesBlocking(t, nodeSlice...)
	defer func() {
		for _, n := range nodeSlice {
			n.Stop()
		}
	}()

	n("A").AddMessage("Hi!")
	<- time.After(5*time.Second)
	n("B").AddMessage("Hi there!")
	n("C").AddMessage("Hi there!")
	n("D").AddMessage("Hi there!")
	n("E").AddMessage("Hi there!")
	<- time.After(5*time.Second)

	// assert
	rtA := n("A").GetRoutingTable()
	require.Contains(t, rtA, nodeId["B"])
	require.Equal(t, nodeAddr["B"], rtA[nodeId["B"]].NextHop)
	require.Contains(t, rtA, nodeId["C"])
	require.Equal(t, nodeAddr["C"], rtA[nodeId["C"]].NextHop)
	require.Contains(t, rtA, nodeId["D"])
	require.Equal(t, nodeAddr["B"], rtA[nodeId["D"]].NextHop)
	require.Contains(t, rtA, nodeId["E"])
	require.Equal(t, nodeAddr["B"], rtA[nodeId["E"]].NextHop)

	rtD := n("B").GetRoutingTable()
	require.Contains(t, rtD, nodeId["A"])
	require.Equal(t, nodeAddr["A"], rtD[nodeId["A"]].NextHop)
	require.Contains(t, rtD, nodeId["C"])
	require.Equal(t, nodeAddr["A"], rtD[nodeId["C"]].NextHop)
	require.Contains(t, rtD, nodeId["D"])
	require.Equal(t, nodeAddr["D"], rtD[nodeId["D"]].NextHop)
	require.Contains(t, rtD, nodeId["E"])
	require.Equal(t, nodeAddr["E"], rtD[nodeId["E"]].NextHop)
}

// -----------------------------------------------------------------------------
// Utility functions

func createNode(t *testing.T, name string, antiEntropy int,
	routeTimer int) (BaseGossiper, string) {

	addr := fmt.Sprintf("127.0.0.1:%v", getRandomPort())
	fullName := fmt.Sprintf("%v---%v", name, t.Name())
	node, err := factory.New(addr, fullName, antiEntropy, routeTimer)
	require.NoError(t, err)
	require.Len(t, node.GetNodes(), 0)
	require.Equal(t, fullName, node.GetIdentifier())

	return node, addr
}

// addAddresses takes any number of node addresses that the gossiper can
// contact in the gossiping network.
func addAddresses(t *testing.T, node BaseGossiper, addresses ...string) {
	numNodes := len(node.GetNodes())
	err := node.AddAddresses(addresses...)
	require.Nil(t, err, "Could not add addresses")

	// Assumes we're not adding twice the same address.
	require.Len(t, node.GetNodes(), numNodes + len(addresses))
}

// startNodesBlocking waits until the node is started
func startNodesBlocking(t *testing.T, nodes ...BaseGossiper) {
	wg := new(sync.WaitGroup)
	wg.Add(len(nodes))
	for idx, _ := range nodes {
		go func(i int) {
			defer wg.Done()
			ready := make(chan struct{})
			go nodes[i].Run(ready)
			<- ready
		}(idx)
	}
	wg.Wait()
}

func streamIncomingGossips(node BaseGossiper) chan GossipPacket {
	ch := make(chan GossipPacket)
	node.RegisterCallback(
		func(origin string, message GossipPacket) {
			ch <- message
		})
	return ch
}

func waitStatusMsg(t *testing.T, ctx context.Context,
	ch <- chan CallbackPacket) *CallbackPacket {
	for {
		select {
		case <-ctx.Done():
			require.Fail(t, "Timed out on reception")
		case p := <-ch:
			if p.Msg.Status != nil {
				return &p
			}
		}
	}
}

func waitRumorMsg(t *testing.T, ctx context.Context,
	ch <- chan CallbackPacket) *CallbackPacket {
	for {
		select {
		case <- ctx.Done():
			require.Fail(t, "Timed out on reception")
		case p := <- ch:
			if p.Msg.Rumor != nil {
				return &p
			}
		}
	}
}

func waitChannel(t *testing.T, ctx context.Context, c <-chan struct{}) {
	select {
	case <- ctx.Done():
		require.Fail(t, "Timed out on reception")
	case <- c:
	}
}

// getRandomPort returns a random port that is not used at the time of testing.
func getRandomPort() string {
	var uiPortStr string

	for {
		uiPort := rand.Intn(65534)
		uiPortStr = strconv.Itoa(uiPort)

		ln, err := net.Listen("tcp", ":"+uiPortStr)

		// If we can listen, that means the port is free
		if err == nil {
			if err := ln.Close() ; err == nil {
				break
			}
		}
	}

	return uiPortStr
}