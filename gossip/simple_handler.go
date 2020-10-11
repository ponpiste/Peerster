// ========== CS-438 HW1 Skeleton ===========
// *** Implement here the handler for simple message processing ***

package gossip

import (
	"net"
	"golang.org/x/xerrors"
	"go.dedis.ch/onet/v3/log"
)

// Exec is the function that the gossiper uses to execute the handler for a SimpleMessage
// processSimple processes a SimpleMessage as such:
// - add message's relay address to the known peers
// - update the relay field
func (msg *SimpleMessage) Exec(g *Gossiper, addr *net.UDPAddr) error {

	var new_msg = SimpleMessage {
		OriginPeerName: msg.OriginPeerName,
		RelayPeerAddr: g.addr,
		Contents: msg.Contents,
	}

	var packet = GossipPacket {
		Simple: &new_msg,
	}

	// the callback might block or be very long
	if g.callback != nil {
		go g.callback(msg.OriginPeerName, packet)
	}

	// asynchronous because the Run()
	// method wants to go back to
	// listening to new messages

	// Todo: call the watcher here or
	// not ?
	go g.broadcast(packet, msg.RelayPeerAddr)

	// The call is synchronous because
	// later in this thread we want to
	// print the list of peers which
	// should include this address
	
	// addAddr does not need to resolve
	// and is fast(er)
	g.addAddress(addr)
	return nil
}

// Exec is the function that the gossiper uses to execute the handler for a RumorMessage
func (msg *RumorMessage) Exec(g *Gossiper, addr *net.UDPAddr) error {

	// if message not received before (lower , equal, greater than current sequence number)
	//   send to random peer (blacklist the current one)
	//   add the current peer to list

	var packet = GossipPacket {
		Rumor: msg,
	}

	// Todo: what to do when sequence 
	// number is strictly greater than
	// last seq number

	latest := g.getLatest(msg.Origin)
	if latest + 1 != msg.ID {return nil}

	// the callback might block or be very long
	if g.callback != nil {
		go g.callback(msg.Origin, packet)
	}

	// Todo factor sendRumor
	// in a function

	// Todo: ask if the blacklist
	// is necessary, can we send 
	// back the gossip to the same 
	// guy (would cause a loop)
	receiver := g.randomPeer(addr.String())
	if receiver != nil {

		// asynchronous because the Run()
		// method wants to go back to
		// listening to new messages
		go g.send(packet, receiver)

		// Todo: mutex ?
		// Todo: use ID or IP
		// Todo: what id there are
		// more than 1 rumor buffered
		// How to identify rumors ?
		// use a queue
		// use ticker, select ... case

		/*

		go func {

			select
				case done
					return
				case timeout 
					msg.Exec()
		}

		*/

		// g.mongering[receiver.String()] = msg

		// go func() {
			
		// 	time.Sleep(10 * time.Second)
		// 	if _, ok := g.mongering[receiver.String()]; ok {

		// 		delete(g.mongering, receiver.String())
		// 	}
		// }()
	}

	g.addMessage(msg.Origin, msg.Text)

	// Todo: make this piece
	// asynchronous, but do not
	// forget mutexes
	want := g.map2slice()
	
	packet = GossipPacket {
		Status: &StatusPacket {
			Want: want,
		},
	}

	go g.send(packet, addr)

	// Todo: make sure it is the addr
	// and not the origin of the message
	g.addAddress(addr)

	// Todo: launch the timer
	// if nothing received after
	// 10 seconds

	// add it to the queue maybe ?
	// idk

	return nil
}

// Exec is the function that the gossiper uses to execute the handler for a StatusMessage
func (msg *StatusPacket) Exec(g *Gossiper, addr *net.UDPAddr) error {
	
	// Todo: what happens if
	// S has new messages AND 
	// R has new messages

	g.addAddress(addr)

	var mp = make(map[string]uint32)
	var needed = false

	for _, i := range msg.Want {

		val, ok := g.messages[i.Identifier]
		if !ok || uint32(len(val) + 1) < i.NextID {
			needed = true
		}

		mp[i.Identifier] = i.NextID
	}

	// receiver has other new messages
	if needed {

		// Todo: make this piece
		// asynchronous, but do not
		// forget mutexes
		want := g.map2slice()
		
		packet := GossipPacket {
			Status: &StatusPacket {
				Want: want,
			},
		}

		go g.send(packet, addr)
	}

	var has int = 0

	// Todo: lock the messages ???
	for key, value := range g.messages {

		var start uint32 = 1
		v, ok := mp[key]

		if ok {
			start = v

			// Sanity check
			// Might happen sometimes
			if v == 1 {
				log.Error("v is equal to 1 in status exec")
			}
		}

		for i := start; i < uint32(len(value) + 1); i++ {

			has++;
			
			var packet = GossipPacket {
				Rumor: &RumorMessage {
					Origin: g.identifier,
					ID: i,
					Text: value[i - 1],
				},
			}

			// asynchronous because the Run()
			// method wants to go back to
			// listening to new messages
			go g.send(packet, addr)
		}
	}

	if has == 0 && !needed && g.ran.Intn(2) == 0 {

		// Todo: how to remember the gossip msg ???
		// if entry in the mongo table

		// packet := GossipPacket{}

		// receiver := g.randomPeer(addr.String())

		// // Might happen once a day
		// if receiver == nil {
		// 	log.Error("No receiver found")
		// 	return xerrors.Errorf("No receiver found")
		// }

		// // asynchronous because the Run()
		// // method wants to go back to
		// // listening to new messages
		// go g.send(packet, receiver)
	}

	return nil
}

// Exec is the function that the gossiper uses to execute the handler for a PrivateMessage
func (msg *PrivateMessage) Exec(g *Gossiper, addr *net.UDPAddr) error {
	return xerrors.Errorf("Implement me")
}
