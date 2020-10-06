// ========== CS-438 HW0 Skeleton ===========
// *** Do not change this file ***

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"sync"

	"go.dedis.ch/cs438/hw1/client"
	"go.dedis.ch/cs438/hw1/gossip"

	"go.dedis.ch/onet/v3/log"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Controller is responsible to be the glue between the gossiping protocol and
// the ui, dispatching responses and messages etc
type Controller struct {
	sync.Mutex
	uiAddress     string
	identifier    string
	gossipAddress string
	gossiper      gossip.BaseGossiper
	cliConn       net.Conn
	messages      []CtrlMessage
	// simpleMode: true if the gossiper should broadcast messages from clients as SimpleMessages
	simpleMode bool
}

type CtrlMessage struct {
	Origin string
	ID     uint32
	Text   string
}

// NewController returns the controller that sets up the gossiping state machine
// as well as the web routing. It uses the same gossiping address for the
// identifier.
func NewController(identifier, uiAddress, gossipAddress string, simpleMode bool,
	g gossip.BaseGossiper, addresses ...string) *Controller {

	c := &Controller{
		identifier:    identifier,
		uiAddress:     uiAddress,
		gossipAddress: gossipAddress,
		simpleMode:    simpleMode,
		gossiper:      g,
	}

	g.RegisterCallback(c.NewMessage)

	return c
}

// Run ...
func (c *Controller) Run() {
	r := mux.NewRouter()
	r.Methods("GET").Path("/message").HandlerFunc(c.GetMessage)
	r.Methods("POST").Path("/message").HandlerFunc(c.PostMessage)
	r.Methods("GET").Path("/origin").HandlerFunc(c.GetDirectNode)
	r.Methods("GET").Path("/node").HandlerFunc(c.GetNode)
	r.Methods("POST").Path("/node").HandlerFunc(c.PostNode)
	r.Methods("GET").Path("/id").HandlerFunc(c.GetIdentifier)
	r.Methods("POST").Path("/id").HandlerFunc(c.SetIdentifier)
	r.PathPrefix("/").Handler(http.FileServer(http.Dir("./static/")))
	loggedRouter := handlers.LoggingHandler(os.Stdout, r)

	server := &http.Server{Addr: c.uiAddress, Handler: loggedRouter}

	err := server.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

// GET /message returns all messages seen so far as json encoded Message
// XXX lot of optimizations to be done here
func (c *Controller) GetMessage(w http.ResponseWriter, r *http.Request) {
	c.Lock()
	defer c.Unlock()
	log.Lvl1("These are the msg", c.messages)
	if err := json.NewEncoder(w).Encode(c.messages); err != nil {
		log.Error(err)
		http.Error(w, "could not encode json", http.StatusInternalServerError)
		return
	}
	log.Lvl1("GUI request for the messages received by the gossiper")
	w.WriteHeader(http.StatusOK)
}

// POST /message with text in the body as raw string
func (c *Controller) PostMessage(w http.ResponseWriter, r *http.Request) {
	c.Lock()
	defer c.Unlock()

	text, ok := readString(w, r)
	if !ok {
		log.Error("Error", ok)
		return
	}

	message := client.ClientMessage{}
	err := json.Unmarshal([]byte(text), &message)
	if err != nil {
		log.Error(err)
		return
	}

	log.Lvl1("the controller received a UI message \"", message.Contents, "\"")

	if c.simpleMode {
		c.gossiper.AddSimpleMessage(message.Contents)
		c.messages = append(c.messages, CtrlMessage{c.identifier, 0, message.Contents})
	} else {
		if message.Destination != "" {
			c.gossiper.AddPrivateMessage(message.Contents, message.Destination, c.gossiper.GetIdentifier(), 10)
			c.messages = append(c.messages, CtrlMessage{c.identifier, 0, message.Contents})
		} else {

			id := c.gossiper.AddMessage(message.Contents)
			c.messages = append(c.messages, CtrlMessage{c.identifier, id, message.Contents})
		}

	}

	w.WriteHeader(200)
}

// GET /node returns list of nodes as json encoded slice of string
func (c *Controller) GetNode(w http.ResponseWriter, r *http.Request) {
	hosts := c.gossiper.GetNodes()
	if err := json.NewEncoder(w).Encode(hosts); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
}

// GET /origin returns list of nodes in the routing table as json encoded slice of string
func (c *Controller) GetDirectNode(w http.ResponseWriter, r *http.Request) {
	hosts := c.gossiper.GetDirectNodes()
	if err := json.NewEncoder(w).Encode(hosts); err != nil {
		log.Error(err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(200)
}

// POST /node with address of node in the body as a string
func (c *Controller) PostNode(w http.ResponseWriter, r *http.Request) {
	text, ok := readString(w, r)
	if !ok {
		return
	}
	log.Lvl1("GUI add node", text)
	if err := c.gossiper.AddAddresses(text); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(200)
}

// GET /id returns the identifier as a raw string in the body
func (c *Controller) GetIdentifier(w http.ResponseWriter, r *http.Request) {
	id := c.gossiper.GetIdentifier()
	if _, err := w.Write([]byte(id)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	log.Lvl1("GUI identifier request")
	w.WriteHeader(200)
}

// POST /id reads the identifier as a raw string in the body and sets the
// gossiper.
func (c *Controller) SetIdentifier(w http.ResponseWriter, r *http.Request) {
	id, ok := readString(w, r)
	if !ok {
		return
	}
	log.Lvl1("GUI set identifier")
	fmt.Println("gui set identifier")
	c.gossiper.SetIdentifier(id)
	w.WriteHeader(200)
}

// NewMessage ...
func (c *Controller) NewMessage(origin string, msg gossip.GossipPacket) {
	c.Lock()
	defer c.Unlock()

	if msg.Rumor != nil {

		c.messages = append(c.messages, CtrlMessage{msg.Rumor.Origin, msg.Rumor.ID, msg.Rumor.Text})
	}
	if msg.Simple != nil {

		c.messages = append(c.messages, CtrlMessage{msg.Simple.OriginPeerName, 0, msg.Simple.Contents})
	}
	log.Lvl1("messages", c.messages)
}

func readString(w http.ResponseWriter, r *http.Request) (string, bool) {
	buff, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "could not read message", http.StatusBadRequest)
		return "", false
	}
	return string(buff), true

}

func Error(args ...interface{}) {
	fmt.Println(append([]interface{}{"ERROR (", "): "}, args...)...)
}
