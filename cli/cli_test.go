package main

import (
	"go.dedis.ch/cs438/peerster/hw0/client"
	"go.dedis.ch/cs438/peerster/hw0/gossip"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/onet/v3/log"
)

func TestCli(t *testing.T) {
	// This test checks that the "sendMsg" function performs the right http
	// call.

	msg := client.ClientMessage{
		Contents: "hi",
	}

	received := received{}

	r := mux.NewRouter()
	r.Methods("POST").Path("/message").HandlerFunc(PostMessageHandler(&received))
	go http.ListenAndServe("127.0.0.1:2001", r)

	time.Sleep(time.Millisecond * 300)
	sendMsg("http://127.0.0.1:2001", &msg)
	time.Sleep(time.Millisecond * 300)

	require.Len(t, received.m, 1)
	require.Equal(t, received.m[0].Contents, "hi")

}

// -----------------------------------------------------------------------------
// Utility functions

func PostMessageHandler(received *received) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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

		log.Lvl1("The controller received an UI message \"", message.Contents, "\"")

		received.m = append(received.m, message)
		w.WriteHeader(200)
	}
}

func readString(w http.ResponseWriter, r *http.Request) (string, bool) {
	buff, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "could not read message", http.StatusBadRequest)
		return "", false
	}
	return string(buff), true

}

type received struct {
	m []client.ClientMessage
}

type packet struct {
	origin  string
	message gossip.GossipPacket
}
