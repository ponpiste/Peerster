// ========== CS-438 HW0 Skeleton ===========
// *** Do not change this file ***

package gossip

// GetFactory returns the Gossip factory
func GetFactory() GossipFactory {
	return BaseGossipFactory{}
}

// GossipPacket defines the packet that gets encoded or deserialized from the
// network.
type GossipPacket struct {
	Simple *SimpleMessage `json:"simple"`
}

// SimpleMessage is a structure for the simple message
type SimpleMessage struct {
	OriginPeerName string `json:"originPeerName"`
	RelayPeerAddr  string `json:"relayPeerAddr"`
	Contents       string `json:"contents"`
}

// NewMessageCallback is the type of function that users of the library should
// provide to get a feedback on new messages detected in the gossip network.
type NewMessageCallback func(origin string, message GossipPacket)

// GossipFactory provides the primitive to instantiate a new Gossiper
type GossipFactory interface {
	New(address, identifier string) (BaseGossiper, error)
}

// BaseGossiper ...
type BaseGossiper interface {
	// GetNodes returns the list of nodes this gossiper knows currently in the
	// network.
	GetNodes() []string
	// SetIdentifier changes the identifier sent with messages originating from this
	// gossiper.
	SetIdentifier(id string)
	// GetIdentifier returns the currently used identifier for outgoing messages from
	// this gossiper.
	GetIdentifier() string
	// AddSimpleMessage takes a text that will be spread through the gossip network
	// with the identifier of g. It returns the ID of the message
	AddSimpleMessage(text string)
	// AddAddresses takes any number of node addresses that the gossiper can contact
	// in the gossiping network.
	AddAddresses(addresses ...string) error
	// RegisterCallback ...
	RegisterCallback(NewMessageCallback)
	// Run creates the UPD connection and starts the gossiper. This function is
	// assumed to be blocking until Stop is called. The ready chan should be
	// closed when the Gossiper is started.
	Run(ready chan struct{})
	// Stop stops the Gossiper
	Stop()
}
