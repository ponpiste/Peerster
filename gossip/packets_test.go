package gossip

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var factory = GetFactory()

func TestGossip_Init(t *testing.T) {
	// We test the creation of different Gossipers, the AddAddresses, GetNode,
	// SetIdentifier and GetIdentifier.

	g1, err := factory.New("127.0.0.1:2001", "g")
	require.NoError(t, err)

	g2, err := factory.New("127.0.0.1:2002", "g2")
	require.NoError(t, err)

	require.Len(t, g1.GetNodes(), 0)

	g1.AddAddresses("127.0.0.1:2002")
	require.Len(t, g1.GetNodes(), 1)
	require.Equal(t, "127.0.0.1:2002", g1.GetNodes()[0])

	g1.AddAddresses("127.0.0.1:2003")
	require.Len(t, g1.GetNodes(), 2)
	require.Equal(t, "127.0.0.1:2003", g1.GetNodes()[1])

	g2.AddAddresses("127.0.0.1:2001")
	require.Len(t, g2.GetNodes(), 1)
	require.Equal(t, "127.0.0.1:2001", g2.GetNodes()[0])

	require.Equal(t, "g", g1.GetIdentifier())
	require.Equal(t, "g2", g2.GetIdentifier())

	g1.SetIdentifier("g1")
	require.Equal(t, "g1", g1.GetIdentifier())
}

func TestGossip_Chain(t *testing.T) {
	// We test the sending of a message in a "chain" fashion, like the
	// following:
	//
	// G1 --> G2 --> G3 --> G4
	//
	// If G1 sends a message, we expect every node to receive it, because G2
	// should relay it to G3. We also expect G2 and G3 to update their list of
	// know nodes.

	g1, receivedG1 := initGossip(t, "127.0.0.1:2001", "g1", "127.0.0.1:2002")
	defer g1.Stop()
	g2, receivedG2 := initGossip(t, "127.0.0.1:2002", "g2", "127.0.0.1:2003")
	defer g2.Stop()
	g3, receivedG3 := initGossip(t, "127.0.0.1:2003", "g3", "127.0.0.1:2004")
	defer g3.Stop()
	g4, receivedG4 := initGossip(t, "127.0.0.1:2004", "g4")
	defer g4.Stop()

	// G1 --> G2 --> G3 --> G4
	g1.AddSimpleMessage("message1")
	// We wait a bit to give a chance for the nodes to receive the message
	time.Sleep(time.Millisecond * 300)

	require.Len(t, receivedG1.p, 0)
	require.Len(t, receivedG2.p, 1)
	require.Len(t, receivedG3.p, 1)
	require.Len(t, receivedG4.p, 1)

	expected := GossipPacket{
		&SimpleMessage{
			OriginPeerName: "g1",
			RelayPeerAddr:  "127.0.0.1:2002",
			Contents:       "message1",
		},
	}

	require.Equal(t, expected, receivedG2.p[0].message)
	require.Equal(t, "g1", receivedG2.p[0].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2003"
	require.Equal(t, expected, receivedG3.p[0].message)
	require.Equal(t, "g1", receivedG3.p[0].origin)

	expected.Simple.RelayPeerAddr = "127.0.0.1:2004"
	require.Equal(t, expected, receivedG4.p[0].message)
	require.Equal(t, "g1", receivedG4.p[0].origin)

	// G2 should've added G1 to its list of know nodes
	require.Len(t, g2.GetNodes(), 2)
	require.Equal(t, "127.0.0.1:2001", g2.GetNodes()[1])

	// G3 should've added G2 to its list of know nodes
	require.Len(t, g3.GetNodes(), 2)
	require.Equal(t, "127.0.0.1:2002", g3.GetNodes()[1])

	// G4 should've added G3 to its list of know nodes
	require.Len(t, g4.GetNodes(), 1)
	require.Equal(t, "127.0.0.1:2003", g4.GetNodes()[0])
}

func TestGossip_ChainSplit(t *testing.T) {
	// We test the sending of a message in a "split chain" fashion, like the
	// following:
	//
	// G1 --> G2 --> G3
	//        G4 --> G5
	//           --> G6 --> G7
	//

	g1, receivedG1 := initGossip(t, "127.0.0.1:2001", "g1", "127.0.0.1:2002", "127.0.0.1:2004")
	defer g1.Stop()
	g2, receivedG2 := initGossip(t, "127.0.0.1:2002", "g2", "127.0.0.1:2003")
	defer g2.Stop()
	g3, receivedG3 := initGossip(t, "127.0.0.1:2003", "g3")
	defer g3.Stop()
	g4, receivedG4 := initGossip(t, "127.0.0.1:2004", "g4", "127.0.0.1:2005", "127.0.0.1:2006")
	defer g4.Stop()
	g5, receivedG5 := initGossip(t, "127.0.0.1:2005", "g5")
	defer g5.Stop()
	g6, receivedG6 := initGossip(t, "127.0.0.1:2006", "g6", "127.0.0.1:2007")
	defer g6.Stop()
	g7, receivedG7 := initGossip(t, "127.0.0.1:2007", "g7")
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

type packet struct {
	origin  string
	message GossipPacket
}

type history struct {
	p []packet
}

func initGossip(t *testing.T, addr, id string, knowNodes ...string) (BaseGossiper, *history) {
	g, err := factory.New(addr, id)
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
