package gossip

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var binfactory = NewBinGossipFactory("./hw0")

func TestBinGossiper_Scenario(t *testing.T) {

	g1, receivedG1 := initBinGossip(t, "127.0.0.1:2001", "g1", "127.0.0.1:2002")
	defer g1.Stop()
	g2, receivedG2 := initBinGossip(t, "127.0.0.1:2002", "g2", "127.0.0.1:2003")
	defer g2.Stop()
	g3, receivedG3 := initBinGossip(t, "127.0.0.1:2003", "g3")
	defer g3.Stop()

	// G1 --> G2 --> {G3}
	g3.AddSimpleMessage("simple message")
	time.Sleep(time.Millisecond * 300)

	require.Len(t, receivedG1.p, 0)
	require.Len(t, receivedG2.p, 0)
	require.Len(t, receivedG3.p, 0)

	// G1 --> {G2} --> G3
	g2.AddSimpleMessage("hi from g2")
	time.Sleep(time.Millisecond * 300)

	require.Len(t, receivedG1.p, 0)
	require.Len(t, receivedG2.p, 0)
	require.Len(t, receivedG3.p, 1)

	expected := GossipPacket{
		Simple: &SimpleMessage{
			OriginPeerName: "g2",
			RelayPeerAddr:  "127.0.0.1:2003",
			Contents:       "hi from g2",
		},
	}

	require.Equal(t, expected, receivedG3.p[0].message)
	require.Equal(t, "g2", receivedG3.p[0].origin)

	nodes := g3.GetNodes()
	require.Len(t, nodes, 1)
	require.Equal(t, "127.0.0.1:2002", nodes[0])

	// {G1} --> G2 <--> G3
	g1.AddSimpleMessage("hi from g1")
	time.Sleep(time.Millisecond * 300)

	require.Len(t, receivedG1.p, 0)
	require.Len(t, receivedG2.p, 1)
	require.Len(t, receivedG3.p, 2)

	expected = GossipPacket{
		Simple: &SimpleMessage{
			OriginPeerName: "g1",
			RelayPeerAddr:  "127.0.0.1:2002",
			Contents:       "hi from g1",
		},
	}

	require.Equal(t, expected, receivedG2.p[0].message)
	require.Equal(t, "g1", receivedG2.p[0].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2003"
	require.Equal(t, expected, receivedG3.p[1].message)
	require.Equal(t, "g1", receivedG3.p[1].origin)
}

func TestBinGossiper_GetNodes_AddAddresses(t *testing.T) {
	g, _ := initBinGossip(t, "127.0.0.1:0", "g")
	defer g.Stop()

	nodes := g.GetNodes()
	require.Len(t, nodes, 0, nodes)

	err := g.AddAddresses("127.0.0.1:4000", "127.0.0.1:4001")
	require.NoError(t, err)
	time.Sleep(time.Millisecond * 300)

	nodes = g.GetNodes()
	require.Len(t, nodes, 2)
	require.Equal(t, "127.0.0.1:4000", nodes[0])
	require.Equal(t, "127.0.0.1:4001", nodes[1])
}

func TestBinGossiper_Get_Set_Identifier(t *testing.T) {
	g, _ := initBinGossip(t, "127.0.0.1:0", "g")
	defer g.Stop()

	id := g.GetIdentifier()
	require.Equal(t, "g", id)

	g.SetIdentifier("new")
	time.Sleep(time.Millisecond * 300)

	id = g.GetIdentifier()
	require.Equal(t, "new", id)
}

// -----------------------------------------------------------------------------
// Integration tests

func TestBinGossip_ChainSplit(t *testing.T) {
	// We test the sending of a message in a "split chain" fashion, like the
	// following, with * denoting a binary gossiper:
	//
	// G1 --> G2* --> G3
	//        G4* --> G5
	//           --> G6 --> G7*
	//

	g1, receivedG1 := initGossip(t, "127.0.0.1:2001", "g1", "127.0.0.1:2002", "127.0.0.1:2004")
	defer g1.Stop()
	g2, receivedG2 := initBinGossip(t, "127.0.0.1:2002", "g2", "127.0.0.1:2003")
	defer g2.Stop()
	g3, receivedG3 := initGossip(t, "127.0.0.1:2003", "g3")
	defer g3.Stop()
	g4, receivedG4 := initBinGossip(t, "127.0.0.1:2004", "g4", "127.0.0.1:2005", "127.0.0.1:2006")
	defer g4.Stop()
	g5, receivedG5 := initGossip(t, "127.0.0.1:2005", "g5")
	defer g5.Stop()
	g6, receivedG6 := initGossip(t, "127.0.0.1:2006", "g6", "127.0.0.1:2007")
	defer g6.Stop()
	g7, receivedG7 := initBinGossip(t, "127.0.0.1:2007", "g7")
	defer g7.Stop()

	// G7 --> .
	//
	// If G7 send a message, nothing should happen because its list of node is
	// empty.
	g7.AddSimpleMessage("void message")
	time.Sleep(time.Millisecond * 300)

	require.Len(t, receivedG1.p, 0)
	require.Len(t, receivedG2.p, 0)
	require.Len(t, receivedG3.p, 0)
	require.Len(t, receivedG4.p, 0)
	require.Len(t, receivedG5.p, 0)
	require.Len(t, receivedG6.p, 0)
	require.Len(t, receivedG7.p, 0)

	// G6 --> G7
	//
	// Only G7 should receive a message
	g6.AddSimpleMessage("message1")
	time.Sleep(time.Millisecond * 300)

	require.Len(t, receivedG1.p, 0)
	require.Len(t, receivedG2.p, 0)
	require.Len(t, receivedG3.p, 0)
	require.Len(t, receivedG4.p, 0)
	require.Len(t, receivedG5.p, 0)
	require.Len(t, receivedG6.p, 0)
	require.Len(t, receivedG7.p, 1)

	expected := GossipPacket{
		&SimpleMessage{
			OriginPeerName: "g6",
			RelayPeerAddr:  "127.0.0.1:2007",
			Contents:       "message1",
		},
	}

	require.Equal(t, expected, receivedG7.p[0].message)
	require.Equal(t, "g6", receivedG7.p[0].origin)

	// {G1} --> G2 --> G3
	//          G4 --> G5
	//             --> G6 <-> G7
	//
	// If G1 sends a message, every node should receive it
	g1.AddSimpleMessage("message2")
	time.Sleep(time.Millisecond * 300)

	require.Len(t, receivedG1.p, 0)
	require.Len(t, receivedG2.p, 1)
	require.Len(t, receivedG3.p, 1)
	require.Len(t, receivedG4.p, 1)
	require.Len(t, receivedG5.p, 1)
	require.Len(t, receivedG6.p, 1)
	require.Len(t, receivedG7.p, 2)

	expected = GossipPacket{
		&SimpleMessage{
			OriginPeerName: "g1",
			RelayPeerAddr:  "127.0.0.1:2002",
			Contents:       "message2",
		},
	}

	require.Equal(t, expected, receivedG2.p[0].message)
	require.Equal(t, "g1", receivedG2.p[0].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2004"
	require.Equal(t, expected, receivedG4.p[0].message)
	require.Equal(t, "g1", receivedG4.p[0].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2005"
	require.Equal(t, expected, receivedG5.p[0].message)
	require.Equal(t, "g1", receivedG5.p[0].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2006"
	require.Equal(t, expected, receivedG6.p[0].message)
	require.Equal(t, "g1", receivedG6.p[0].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2007"
	require.Equal(t, expected, receivedG7.p[1].message)
	require.Equal(t, "g1", receivedG7.p[1].origin)

	// Now sending from G6. It should reach every node because, when we sent a
	// message from G1, every node took the opportunity to add the sender
	// address to its list of known node.
	//
	// G1 <-> G2 <-> G3
	//        G4 <-> G5
	//           <-> {G6} <-> G7
	g6.AddSimpleMessage("message3")
	time.Sleep(time.Millisecond * 300)

	require.Len(t, receivedG1.p, 1)
	require.Len(t, receivedG2.p, 2)
	require.Len(t, receivedG3.p, 2)
	require.Len(t, receivedG4.p, 2)
	require.Len(t, receivedG5.p, 2)
	require.Len(t, receivedG6.p, 1)
	require.Len(t, receivedG7.p, 3)

	expected = GossipPacket{
		&SimpleMessage{
			OriginPeerName: "g6",
			RelayPeerAddr:  "127.0.0.1:2001",
			Contents:       "message3",
		},
	}

	require.Equal(t, expected, receivedG1.p[0].message)
	require.Equal(t, "g6", receivedG1.p[0].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2002"
	require.Equal(t, expected, receivedG2.p[1].message)
	require.Equal(t, "g6", receivedG2.p[1].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2003"
	require.Equal(t, expected, receivedG3.p[1].message)
	require.Equal(t, "g6", receivedG3.p[1].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2004"
	require.Equal(t, expected, receivedG4.p[1].message)
	require.Equal(t, "g6", receivedG4.p[1].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2005"
	require.Equal(t, expected, receivedG5.p[1].message)
	require.Equal(t, "g6", receivedG5.p[1].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2007"
	require.Equal(t, expected, receivedG7.p[2].message)
	require.Equal(t, "g6", receivedG7.p[2].origin)
}

// -----------------------------------------------------------------------------
// Utility functions

func initBinGossip(t *testing.T, addr, id string, knowNodes ...string) (BaseGossiper, *history) {
	g, err := binfactory.New(addr, id)
	require.NoError(t, err)

	g.AddAddresses(knowNodes...)

	h := history{}

	g.RegisterCallback(func(origin string, message GossipPacket) {
		h.p = append(h.p, packet{origin: origin, message: message})
	})

	ready := make(chan struct{})
	go g.Run(ready)
	<-ready

	return g, &h
}
