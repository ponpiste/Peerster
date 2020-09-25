package gossip

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/xerrors"
	"gopkg.in/dedis/onet.v2/log"
)

const callbackEndpoint = "/callback"

// NewBinGossipFactory creates a new bin gossip factory, which build a gossiper
// based on a provided binary.
func NewBinGossipFactory(binPath string) GossipFactory {
	return Factory{
		binPath: binPath,
	}
}

// Factory provides a factory to instantiate a Gossiper
//
// - implements gossip.GossipFactory
type Factory struct {
	binPath string
}

// New implements gossip.GossipFactory.
func (f Factory) New(address, identifier string) (BaseGossiper, error) {

	// The UI port is used to communicate with the gossiper. This is our API
	// endpoint to talk to the gossiper that is embedded in the binary.
	uiPortStr := getRandomPort()

	uiURL, err := url.Parse("http://127.0.0.1:" + uiPortStr)
	if err != nil {
		return nil, xerrors.Errorf("failed to parse uiURL: %v", err)
	}

	g := BinGossiper{
		address:    address,
		identifier: identifier,
		stop:       make(chan struct{}),
		uiURL:      uiURL,
		stopwait:   new(sync.WaitGroup),
	}

	// We start a new callback http server. This server is used for the
	// controller to notify us when a new message arrives. We need it to call
	// the callback registered by the RegisterCallback() function: Each time
	// this server receives a notification (ie. a POST /callback), it can then
	// notify the callback. The controller knows that it must contact this
	// server because we specify it in the binary argument when we launch it.
	callbackAddr, err := setupCallbackSrv(&g, g.stop, g.stopwait)

	log.LLvl1("starting the gossiper and its controller with the binary at", f.binPath)

	err = startBinGossip(f.binPath, uiPortStr, address, identifier, callbackAddr, g.stop, g.stopwait)
	if err != nil {
		return nil, xerrors.Errorf("failed to start binary: %v", err)
	}

	time.Sleep(time.Millisecond * 300)

	return &g, nil
}

// BinGossiper implements a gossiper based on a binary.
//
// - implements gossip.BaseGossiper.
type BinGossiper struct {
	address    string
	identifier string
	callback   NewMessageCallback
	uiURL      *url.URL
	stop       chan struct{}
	stopwait   *sync.WaitGroup
}

// BroadcastMessage implements gossip.BaseGossiper.
func (g BinGossiper) BroadcastMessage(p GossipPacket) {
	panic("not implemented") // TODO: Implement
}

// RegisterHandler implements gossip.BaseGossiper.
func (g BinGossiper) RegisterHandler(handler interface{}) error {
	panic("not implemented") // TODO: Implement
}

// GetNodes implements gossip.BaseGossiper.
func (g BinGossiper) GetNodes() []string {
	URL := g.uiURL.String() + "/node"
	parsedURL, err := url.Parse(URL)
	if err != nil {
		log.Errorf("failed to parse url: %v", err)
		return nil
	}

	req := &http.Request{
		Method: http.MethodGet,
		URL:    parsedURL,
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Error(err)
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("expected 200 response, got: %s", resp.Status)
		return nil
	}

	res := make([]string, 0)

	resBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error(err)
	}

	nullStr := `null
`

	if string(resBuf) == nullStr {
		return []string{}
	}

	err = json.Unmarshal(resBuf, &res)
	if err != nil {
		log.Error(err)
	}

	return res
}

// SetIdentifier implements gossip.BaseGossiper.
func (g BinGossiper) SetIdentifier(id string) {
	URL := g.uiURL.String() + "/id"
	parsedURL, err := url.Parse(URL)
	if err != nil {
		log.Errorf("failed to parse url: %v", err)
		return
	}

	req := &http.Request{
		Method: http.MethodPost,
		URL:    parsedURL,
		Header: map[string][]string{
			"Content-Type": {"text/plain; charset=UTF-8"},
		},
		Body: ioutil.NopCloser(strings.NewReader(id)),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("failed to add address: %v", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("expected 200 response, got: %s", resp.Status)
		return
	}
}

// GetIdentifier implements gossip.BaseGossiper.
func (g BinGossiper) GetIdentifier() string {
	URL := g.uiURL.String() + "/id"
	parsedURL, err := url.Parse(URL)
	if err != nil {
		log.Errorf("failed to parse url: %v", err)
		return ""
	}

	req := &http.Request{
		Method: http.MethodGet,
		URL:    parsedURL,
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("failed to call request: %v", err)
		return ""
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("expected 200 response, got: %s", resp.Status)
		return ""
	}

	resBuf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Errorf("failed to read body: %v", err)
		return ""
	}

	return string(resBuf)
}

// AddSimpleMessage implements gossip.BaseGossiper.
func (g BinGossiper) AddSimpleMessage(text string) {

	values := map[string]string{"contents": text}
	jsonValue, err := json.Marshal(values)
	if err != nil {
		log.Errorf("failed to marshal json: %v", err)
		return
	}

	URL := g.uiURL.String() + "/message"
	parsedURL, err := url.Parse(URL)
	if err != nil {
		log.Errorf("failed to parse url: %v", err)
		return
	}

	req := &http.Request{
		Method: "POST",
		URL:    parsedURL,
		Header: map[string][]string{
			"Content-Type": {"application/json; charset=UTF-8"},
		},
		Body: ioutil.NopCloser(strings.NewReader(string(jsonValue[:]))),
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.Errorf("failed to add simple message: %v", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Errorf("expected 200 response, got: %s", resp.Status)
		return
	}

	log.Lvl1("DONE sending gossip packet to ", g.address, "resp is", resp)
}

// AddAddresses implements gossip.BaseGossiper.
func (g BinGossiper) AddAddresses(addresses ...string) error {
	// since the controller only accept on address at a time, we must send N
	// requests
	URL := g.uiURL.String() + "/node"
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return xerrors.Errorf("failed to parse url: %v", err)
	}

	for _, addr := range addresses {
		req := &http.Request{
			Method: http.MethodPost,
			URL:    parsedURL,
			Header: map[string][]string{
				"Content-Type": {"text/plain; charset=UTF-8"},
			},
			Body: ioutil.NopCloser(strings.NewReader(addr)),
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return xerrors.Errorf("failed to add address: %v", err)
		}

		if resp.StatusCode != http.StatusOK {
			return xerrors.Errorf("expected 200 response, got: %s", resp.Status)
		}
	}

	return nil
}

// RegisterCallback implements gossip.BaseGossiper.
func (g *BinGossiper) RegisterCallback(c NewMessageCallback) {
	g.callback = c
}

// Run implements gossip.BaseGossiper.
func (g BinGossiper) Run(ready chan struct{}) {
	close(ready)
}

// Stop implements gossip.BaseGossiper.
func (g BinGossiper) Stop() {
	close(g.stop)
	g.stopwait.Wait()

	log.LLvl1("process and callback server stopped!")
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
			ln.Close()
			break
		}
	}

	return uiPortStr
}

// setupCallbackSrv starts the callback server and attaches the needed handler
// to it. It also proprely stops it when the stop chan closes.
func setupCallbackSrv(g *BinGossiper, stop chan struct{}, stopwait *sync.WaitGroup) (string, error) {
	stopwait.Add(1)

	mux := http.NewServeMux()
	server := &http.Server{Addr: "127.0.0.1:0", Handler: mux}

	// We create the connection
	ln, err := net.Listen("tcp", server.Addr)
	if err != nil {
		return "", xerrors.Errorf("failed to listen: %v", err)
	}
	log.LLvl1("callback server started at", ln.Addr())

	handleCallbacks := func(w http.ResponseWriter, req *http.Request) {
		buf, err := ioutil.ReadAll(req.Body)
		if err != nil {
			log.Errorf("failed to read buf in callback: %v", err)
			return
		}

		packet := CallbackPacket{}
		err = json.Unmarshal(buf, &packet)
		if err != nil {
			log.Errorf("failed to unmarshal packet: %v", err)
			return
		}

		if g.callback != nil {
			g.callback(packet.Origin, packet.Msg)
		}
	}

	mux.HandleFunc(callbackEndpoint, handleCallbacks)

	wait := make(chan struct{})
	go func() {
		close(wait)
		err := server.Serve(ln)
		if err != nil && err != http.ErrServerClosed {
			log.Errorf("failed to start http server: %v", err)
		}
	}()

	<-wait

	// Routing to wait for the close signal, which tells us to stop the callback
	// server.
	go func() {
		defer stopwait.Done()
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
		defer cancel()
		err := server.Shutdown(ctx)
		if err != nil {
			log.Errorf("failed to stop callback server: %v", err)
		}
	}()

	return ln.Addr().String(), nil
}

// startBinGossip calls the binary that starts the gossiper. It also ensures
// that the process is proprely closed once the stop chan is closed.
func startBinGossip(binPath, uiPortStr, address, id, callbackAddr string,
	stop chan struct{}, stopwait *sync.WaitGroup) error {

	stopwait.Add(1)

	cmd := exec.Command(binPath, "--UIPort", uiPortStr, "--gossipAddr",
		address, "--name", id, "--hookURL",
		"http://"+callbackAddr+callbackEndpoint)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Start()
	if err != nil {
		return xerrors.Errorf("failed to run node: %v", err)
	}

	go func() {
		defer stopwait.Done()

		<-stop
		err := cmd.Process.Signal(os.Interrupt)
		if err != nil {
			log.Errorf("failed to stop process: %v", err)
			err = cmd.Process.Kill()
			if err != nil {
				log.Errorf("failed to kill: %v", err)
				return
			}
		}

		_, err = cmd.Process.Wait()
		if err != nil {
			log.Errorf("failed to wait for stop process: %v", err)
			return
		}
	}()

	return nil
}

// CallbackPacket describes the content of a callback
type CallbackPacket struct {
	Origin string
	Msg    GossipPacket
}
