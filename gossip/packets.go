// ========== CS-438 HW1 Skeleton ===========
// Define the packet structs here.
package gossip

import (
       "context"
)

// GetFactory returns the Gossip factory
func GetFactory() GossipFactory {
	return BaseGossipFactory{}
}

// GossipPacket defines the packet that gets encoded or deserialized from the
// network.
type GossipPacket struct {
	Simple  *SimpleMessage  `json:"simple"`
	Rumor   *RumorMessage   `json:"rumor"`
	Status  *StatusPacket   `json:"status"`
	Private *PrivateMessage `json:"private"`
}

// SimpleMessage is a structure for the simple message
type SimpleMessage struct {
	OriginPeerName string `json:"originPeerName"`
	RelayPeerAddr  string `json:"relayPeerAddr"`
	Contents       string `json:"contents"`
}

// RumorMessage denotes of an actual message originating from a given Peer in the network.
type RumorMessage struct {
	Origin string `json:"origin"`
	ID     uint32 `json:"id"`
	Text   string `json:"text"`
}

// StatusPacket is sent as a status of the current local state of messages seen
// so far. It can start a rumormongering process in the network.
type StatusPacket struct {
	Want []PeerStatus `json:"want"`
}

// PeerStatus shows how far have a node see messages coming from a peer in
// the network.
type PeerStatus struct {
	Identifier string `json:"identifier"`
	NextID     uint32 `json:"nextid"`
}

// RouteStruct to hold the routes to other nodes.
type RouteStruct struct {
	// NextHop is the address of the forwarding peer
	NextHop string
	// LastID is the sequence number
	LastID uint32
}

// PrivateMessage is sent privately to one peer
type PrivateMessage struct {
	Origin      string `json:"origin"`
	ID          uint32 `json:"id"`
	Text        string `json:"text"`
	Destination string `json:"destination"`
	HopLimit    int    `json:"hoplimit"`
}

// CallbackPacket describes the content of a callback
type CallbackPacket struct {
	Addr string
	Msg  GossipPacket
}

// NewMessageCallback is the type of function that users of the library should
// provide to get a feedback on new messages detected in the gossip network.
type NewMessageCallback func(origin string, message GossipPacket)

// GossipFactory provides the primitive to instantiate a new Gossiper
type GossipFactory interface {
	New(address, identifier string, antiEntropy int, routeTimer int) (BaseGossiper, error)
}

// BaseGossiper ...
type BaseGossiper interface {
	BroadcastMessage(GossipPacket)
	RegisterHandler(handler interface{}) error
	// GetNodes returns the list of nodes this gossiper knows currently in the
	// network.
	GetNodes() []string
	// GetDirectNodes returns the list of nodes this gossiper knows  in its routing table
	GetDirectNodes() []string
	// SetIdentifier changes the identifier sent with messages originating from this
	// gossiper.
	SetIdentifier(id string)
	// GetIdentifier returns the currently used identifier for outgoing messages from
	// this gossiper.
	GetIdentifier() string
	// AddSimpleMessage takes a text that will be spread through the gossip network
	// with the identifier of g. It returns the ID of the message
	AddSimpleMessage(text string)
	// AddMessage takes a text that will be spread through the gossip network
	// with the identifier of g. It returns the ID of the message
	AddMessage(text string) uint32
	// AddPrivateMessage
	AddPrivateMessage(text string, dest string, origin string, hoplimit int)
	// AddAddresses takes any number of node addresses that the gossiper can contact
	// in the gossiping network.
	AddAddresses(addresses ...string) error
	// GetRoutingTable returns the routing table of the node.
	GetRoutingTable() map[string]*RouteStruct
	// RegisterCallback registers a callback needed by the controller to update
	// the view.
	RegisterCallback(NewMessageCallback)
	// Run creates the UPD connection and starts the gossiper. This function is
	// assumed to be blocking until Stop is called. The ready chan should be
	// closed when the Gossiper is started.
	Run(ready chan struct{})
	// Stop stops the Gossiper
	Stop()
	// Watch returns a chan that is populated with new incoming packets if
	// fromIncoming is true, otherwise from sent messages.
	Watch(ctx context.Context, fromIncoming bool) <-chan CallbackPacket
}
