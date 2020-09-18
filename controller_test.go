package main

import (
	"bytes"
	"go.dedis.ch/cs438/peerster/hw0/gossip"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestController_Message(t *testing.T) {
	// This scenario checks that the Gossiper can send and receive messages
	// correctly when used with the controller.
	//
	// We instantiate the controller with G1, and use that controller to talk to
	// G2. G1 and G2 know about each other.
	//
	// List of events:
	//
	// Check:
	// GET /message => should return null since G1 hasn't received any message
	//
	// Action:
	// G2 sends a message to G1 (G2 --> G1)
	//
	// Check:
	// GET /message => should contain the message from G2
	//
	// Action:
	// POST /message (that means G1 --> G2)
	//
	// Check:
	// We check that G2 received the message from G1

	factory := gossip.GetFactory()

	g1, err := factory.New("127.0.0.1:2001", "g1")
	require.NoError(t, err)

	g2, err := factory.New("127.0.0.1:2002", "g2")
	require.NoError(t, err)

	g2.AddAddresses("127.0.0.1:2001")

	type packet struct {
		origin  string
		message gossip.GossipPacket
	}

	// To keep track of incoming packets of G2
	receivedG2 := []packet{}
	g2.RegisterCallback(func(origin string, message gossip.GossipPacket) {
		receivedG2 = append(receivedG2, packet{origin: origin, message: message})
	})

	c := NewController("id", "", "127.0.0.1:2001", true, g1)

	// We request the messages on G1, which should return null since there
	// hasn't been any exchanged messages.
	rr := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/message", nil)
	require.NoError(t, err)

	c.GetMessage(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)

	res, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err)
	require.Equal(t, `null
`, string(res))

	// We start G1 and G2
	ready := make(chan struct{})
	go g1.Run(ready)
	defer g1.Stop()
	<-ready

	ready = make(chan struct{})
	go g2.Run(ready)
	defer g2.Stop()
	<-ready

	// G2 --> G1
	g2.AddSimpleMessage("hi from g2")
	time.Sleep(time.Millisecond * 300)

	// We expect now G1 to return the message sent from G2
	c.GetMessage(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)

	res, err = ioutil.ReadAll(rr.Body)
	require.NoError(t, err)
	require.Equal(t, `[{"Origin":"g2","Text":"hi from g2"}]
`, string(res))

	// G1 sends a message using the POST action
	var requestBody bytes.Buffer
	requestBody.WriteString(`{"contents":"hi from g1"}`)

	req, err = http.NewRequest(http.MethodPost, "/message", &requestBody)
	require.NoError(t, err)

	c.PostMessage(rr, req)
	time.Sleep(time.Millisecond * 300)

	// We check that G2 got the message
	require.Len(t, receivedG2, 1)

	expected := gossip.GossipPacket{
		Simple: &gossip.SimpleMessage{
			OriginPeerName: "g1",
			RelayPeerAddr:  "127.0.0.1:2002",
			Contents:       "hi from g1",
		},
	}

	require.Equal(t, expected, receivedG2[0].message)
}

func TestController_Node(t *testing.T) {
	// This test checks that a Gossiper can update its list of nodes via the two
	// http calls:
	// - GET /node
	// - PUT /node

	factory := gossip.GetFactory()

	g1, err := factory.New("127.0.0.1:2001", "g1")
	require.NoError(t, err)

	c := NewController("id", "", "127.0.0.1:2001", true, g1)

	// We request the list of known nodes, which should be empty
	rr := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/node", nil)
	require.NoError(t, err)

	c.GetNode(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)

	res, err := ioutil.ReadAll(rr.Body)
	require.NoError(t, err)
	require.Equal(t, `[]
`, string(res))

	// We start G1
	ready := make(chan struct{})
	go g1.Run(ready)
	defer g1.Stop()
	<-ready

	// We add a node to G1 and check that GET /node returns the node
	g1.AddAddresses("127.0.0.1:2002")

	c.GetNode(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)

	res, err = ioutil.ReadAll(rr.Body)
	require.NoError(t, err)
	require.Equal(t, `["127.0.0.1:2002"]
`, string(res))

	// Send a PUT request to add a node
	var requestBody bytes.Buffer
	requestBody.WriteString(`127.0.0.1:2003`)

	req, err = http.NewRequest(http.MethodPut, "/node", &requestBody)
	require.NoError(t, err)

	c.PostNode(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)

	// Check the new list of nodes, which should contain two entries now
	req, err = http.NewRequest(http.MethodGet, "/node", nil)
	require.NoError(t, err)

	c.GetNode(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)

	res, err = ioutil.ReadAll(rr.Body)
	require.NoError(t, err)
	require.Equal(t, `["127.0.0.1:2002","127.0.0.1:2003"]
`, string(res))
}

func TestController_Identifier(t *testing.T) {
	// This test checks that a Gossiper can update its identifier via the two
	// http calls:
	// - GET /id
	// - POST /id

	factory := gossip.GetFactory()

	g1, err := factory.New("127.0.0.1:2001", "g1")
	require.NoError(t, err)

	c := NewController("id", "", "127.0.0.1:2001", true, g1)

	// We request the identifier
	rr := httptest.NewRecorder()
	req, err := http.NewRequest(http.MethodGet, "/id", nil)
	require.NoError(t, err)

	c.GetIdentifier(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
	require.Equal(t, "g1", rr.Body.String())

	// Update the identifier with a POST
	var requestBody bytes.Buffer
	requestBody.WriteString(`newID`)

	req, err = http.NewRequest(http.MethodPost, "/node", &requestBody)
	require.NoError(t, err)

	c.SetIdentifier(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)

	// Check the new identifier with a GET
	req, err = http.NewRequest(http.MethodGet, "/id", nil)
	require.NoError(t, err)

	rr.Body.Reset()

	c.GetIdentifier(rr, req)
	require.Equal(t, http.StatusOK, rr.Result().StatusCode)
	require.Equal(t, "newID", rr.Body.String())
}
