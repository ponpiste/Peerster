# CS-438 - Peerster Homework 0

This is the skeleton implementation for the CS-438 Peerster system, homework 0.

## Overview

The homework implements a basic message broadcasting. Every node in the system, called a peer, forwards the message it receives to all its known neighbors except for the node that send it the message. Message transmission takes place over UDP and there is no mechanism in place to prevent message loops. Every peer also maintains a list of known neighbors, which is updated when receiving messages.

For convenience, there are two clients: a CLI one and a GUI one. The CLI is useful for testing and can only send a message to a peer running on the same machine. The GUI one can send messages, displays the messages received from other peers, displays and can update the peer identifier, as well as displays and can modify the list of known peers.

Architecturally, there is a controller component, part of every peer, that glues together the gossiping logic with the client logic. The controller runs a web server that the CLI and the GUI talk to.

## Repo organization

The repo is organized as follows.
- The `main` package contains the controller component, which has the logic for the GUI.
- The `main` package also contains the CLI, which  resides under `cli/`.
- The `gossip` package contains the gossiper logic, found under `gossip/`
- The `client` package, found under `client/`, contains client-related structs
- The `static` folder contains javascript and html GUI files

The skeleton contains the controller code, the GUI code, as well as some structs for the client and the gossiper. In terms of files, the skeleton contains a complete `controller.go`, `main.go`, which you do not need to and should not modify. The skeleton also contains structs and interfaces in `client/packets.go` and `gossip/packets.go`, as well as the handler implementation in `gossip/handlers.go`. There is no need to modify the existing structs, interfaces and functions in these files. The GUI HTML and javascript files are provided in `static`. 

The tests provided are in `controller_test.go`, `gossip/packets_test.go`, `cli/cli_test.go`.

For the CLI implementation, you are expected to fill in the file `cli/cli.go`.

For the gossiper implementation, you are expected to work under the `gossip` directory. The gossiper structs and functionality should be placed in `gossip/gossiper.go`. The handler for message processing should be implemented in `gossip/simple_handler.go`.

## How to build and run the code

`go build` in the root folder

`go build` in the cli folder

Example of how to run the code:

`./peerster -UIPort=2222 -gossipAddr=127.0.0.1:5000 -name=p1 -peers=127.0.0.1:5001` 

`./cli/cli -UIPort=2222 -msg="My great message to the world"`

The GUI can be opened in a browser at `127.0.0.1:2222`

## Unit tests

You can run all the tests with `go test ./... -v`,  
or you can run a particular test, for example from the `gossip` folder: `go test -run TestGossip_Init`.


## Manual test examples

### Test 1

p1 --> p2 --> p3

Initially, p1 has p2 as a known peer, and p2 has p3 as a known peer.
p1 sends a message. The message should reach everyone *at most once* (because of unreliable delivery over UDP). Additionally, if the message is delievered all the way to p3, p2 adds p1 as a known neighbor and p3 adds p2 as a known peer.

To run this test:

`./peerster -UIPort=2222 -gossipAddr=127.0.0.1:5000 -name=p1 -peers=127.0.0.1:5001` 

`./peerster -UIPort=2223 -gossipAddr=127.0.0.1:5001 -name=p2 -peers=127.0.0.1:5002` 

`./peerster -UIPort=2224 -gossipAddr=127.0.0.1:5002 -name=p3

`./cli/cli -UIPort=2222`


### Test 2
 
p1 --> p2 --> p3 --> p1


Circular network. Initially, p1 has p2 as a known peer, p2 has p3 as a known peer, and p3 has p1 as a known peer.
p1 sends a message. The message should reach everyone and loop infinitely through the network, minus the cases when the very first transmission fails because of unreliable delivery over UDP. Additionally, every node adds the other two in their list of known neighbors.


`./peerster -UIPort=2222 -gossipAddr=127.0.0.1:5000 -name=p1 -peers=127.0.0.1:5001` 

`./peerster -UIPort=2223 -gossipAddr=127.0.0.1:5001 -name=p2 -peers=127.0.0.1:5002` 

`./peerster -UIPort=2224 -gossipAddr=127.0.0.1:5002 -name=p3 -peers=127.0.0.1:5000`

`./cli/cli -UIPort=2222`

# Run the integration test locally

The integration test checks that your gossiper can work with a reference implementation that we provide as a binary.

The test is written in `gossip/bingossip_test.go`. There, you will find the same test case as in `packets_test.go`, except that it uses a combination of multiple reference gossipers with your own gossiper implementation. This works by either calling the `initGossip` function, which uses the standard gossip factory, or the `initBinGossip`, which uses the `binfactory` that create a gossiper based on a provided binary.

The only thing you need to do to run the `bingossip_test.go` in local is to provide the correct binary for the binfactory to use. Right now, the factory is looking for `./hw0`, which means that the binary `hw0` is expected to be in the `gossip/` folder. Therefore, be sure that it is there (ie. you have `gossip/hw0`).

If your machine is a MacOs (Darwin) rather than Unix, there's the `gossip/hw0_osx` binary, which you first need to rename to `gossip/hw0` and then run the tests.


Depending on your platform, you may need a different binary that has been compiled for your platform. We provide binaries for MacOS and Linux (64 bits). So, be sure to use the correct version, rename the binary to hw0, and place it in the correct gossip/ folder.
