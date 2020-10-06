# CS-438 - Peerster Homework 1

This is the reference implementation for CS438 Peerster system, homework 1.

## Overview

The homework implements message gossiping, routing and private messaging. Every node in the system, called a peer, gossips the messages using rumormongering and antientropy. Nodes use DSDV protocol for routing private messages.

For convenience, there are two clients: a CLI one and a GUI one. The CLI is useful for testing and can only send a message to a peer running on the same machine. The GUI one can send messages, displays the messages received from other peers, displays and can update the peer identifier, as well as displays and can modify the list of known peers. GUI also supports private messaging.

Architecturally, there is a controller component, part of every peer, that glues together the gossiping logic with the client logic. The controller runs a web server that the CLI and the GUI talk to.


## Repo organization

The repo is organized as follows.
	- The main package contains the controller component, which has the logic for the GUI. The CLI resides under "cli/".
	- The gossip package contains the gossiper logic, found under "gossip/"
	- The client package, found under "client/", contains client-related structs
	- The "static" folder contains javascript and html GUI files


## How to build and run the code

`go build` in the root folder

`go build` in the cli folder

Example of how to run the code:

`./hw1_new -UIPort=2222 -gossipAddr=127.0.0.1:5000 -name=p1 -peers=127.0.0.1:5001 -antiEntropy=20 -broadcast=false -rtimer=20` 

`./cli/cli -UIPort=2222 -msg="P13" -dest="p2"` or `./cli/cli -UIPort=2222 -msg="P13"`

The GUI can be opened in a browser at `127.0.0.1:2222`
 
