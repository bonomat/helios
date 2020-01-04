package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcd/btcec"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	crypto "github.com/libp2p/go-libp2p-crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

// setupHosts is responsible for creating libp2p hosts.
func setupHosts(n int, initialPort int) ([]*libp2pPubSub, []*core.Host) {
	// hosts used in libp2p communications
	hosts := make([]*core.Host, n)
	pubSubs := make([]*libp2pPubSub, n)

	for i := range hosts {

		pubsub := new(libp2pPubSub)

		// creating libp2p hosts
		host := pubsub.createPeer(i, initialPort+i)
		hosts[i] = host
		// creating pubsubs
		pubsub.initializePubSub(*host)
		pubSubs[i] = pubsub
	}
	return pubSubs, hosts
}

// setupNetworkTopology sets up a simple network topology for test.
func setupNetworkTopology(hosts []*core.Host) {

	// Connect hosts to each other in a topology
	// host0 ---- host1 ---- host2 ----- host3 ----- host4
	//	 			|		   				|    	   |
	//	            ------------------------------------
	connectHostToPeer(*hosts[1], getLocalhostAddress(*hosts[0]))
	connectHostToPeer(*hosts[2], getLocalhostAddress(*hosts[1]))
	connectHostToPeer(*hosts[3], getLocalhostAddress(*hosts[2]))
	connectHostToPeer(*hosts[4], getLocalhostAddress(*hosts[3]))
	connectHostToPeer(*hosts[4], getLocalhostAddress(*hosts[1]))
	connectHostToPeer(*hosts[3], getLocalhostAddress(*hosts[1]))
	connectHostToPeer(*hosts[4], getLocalhostAddress(*hosts[1]))
	connectHostToPeer(*hosts[0], "/ip4/10.0.0.9/tcp/9900/p2p/16Uiu2HAkwNtUqDvkanuLN4sdrSwMDHr95CgZ5h7FwYhmxgbG2qAW")

	// Wait so that subscriptions on topic will be done and all peers will "know" of all other peers
	time.Sleep(time.Second * 2)

}

func startListening(pubSubs []*libp2pPubSub, hosts []*core.Host) {
	wg := &sync.WaitGroup{}

	for i, host := range hosts {

		wg.Add(1)
		go func(host *core.Host, pubSub *libp2pPubSub) {
			for true {
				msg, _ := pubSub.subscription.Next(context.Background())
				fmt.Printf("Node %s received Message: '%s' from '%s' \n", (*host).ID().Pretty(), string(msg.Data), string(msg.ReceivedFrom.Pretty()))
			}

		}(host, pubSubs[i])
	}
	fmt.Println("Broadcasting a message on node 0...")
	pubSubs[0].Broadcast("Bananaboatckr")

	fmt.Println("Broadcasting a message on node 1...")
	pubSubs[1].Broadcast("Bananaboatckr 2")
	wg.Wait()
	fmt.Println("The END")
}

func main() {
	n := 5
	initialPort := 9900

	// Create hosts in libp2p
	pubSubs, hosts := setupHosts(n, initialPort)

	defer func() {
		fmt.Println("Closing hosts")
		for _, h := range hosts {
			_ = (*h).Close()
		}
	}()

	setupNetworkTopology(hosts)

	startListening(pubSubs, hosts)

}

type libp2pPubSub struct {
	pubsub       *pubsub.PubSub       // PubSub of each individual node
	subscription *pubsub.Subscription // Subscription of individual node
	topic        string               // PubSub topic
}

// Broadcast Uses PubSub publish to broadcast messages to other peers
func (c *libp2pPubSub) Broadcast(msg string) {
	// Broadcasting to a topic in PubSub
	err := c.pubsub.Publish(c.topic, []byte(msg))
	if err != nil {
		fmt.Printf("Error : %v\n", err)
		return
	}
}

// Receive gets message from PubSub in a blocking way
func (c *libp2pPubSub) Receive() (string, string) {
	// Blocking function for consuming newly received messages
	// We can access message here
	msg, _ := c.subscription.Next(context.Background())
	return string(msg.From), string(msg.Data)
}

// createPeer creates a peer on localhost and configures it to use libp2p.
func (c *libp2pPubSub) createPeer(nodeId int, port int) *core.Host {
	// Creating a node
	h, err := createHost(port)
	if err != nil {
		panic(err)
	}

	// Returning pointer to the created libp2p host
	return &h
}

// initializePubSub creates a PubSub for the peer and also subscribes to a topic
func (c *libp2pPubSub) initializePubSub(h core.Host) {
	var err error
	// Creating pubsub
	// every peer has its own PubSub
	c.pubsub, err = applyPubSub(h)
	if err != nil {
		fmt.Printf("Error : %v\n", err)
		return
	}

	// Registering to the topic
	c.topic = "TOPIC"
	// Creating a subscription and subscribing to the topic
	c.subscription, err = c.pubsub.Subscribe(c.topic)
	if err != nil {
		fmt.Printf("Error : %v\n", err)
		return
	}

}

// createHost creates a host with some default options and a signing identity
func createHost(port int) (core.Host, error) {
	// Producing pirate key
	prvKey, _ := ecdsa.GenerateKey(btcec.S256(), rand.Reader)
	sk := (*crypto.Secp256k1PrivateKey)(prvKey)

	// Starting a peer with default configs
	opts := []libp2p.Option{
		libp2p.ListenAddrStrings(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", port)),
		libp2p.Identity(sk),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
	}

	h, err := libp2p.New(context.Background(), opts...)
	if err != nil {
		return nil, err
	}

	return h, nil
}

// getLocalhostAddress is used for getting address of hosts
func getLocalhostAddress(h core.Host) string {
	for _, addr := range h.Addrs() {
		if strings.Contains(addr.String(), "10.0.0.15") {
			fmt.Println(addr.String() + "/p2p/" + h.ID().Pretty())
			return addr.String() + "/p2p/" + h.ID().Pretty()
		}
	}

	return ""
}

// applyPubSub creates a new GossipSub with message signing
func applyPubSub(h core.Host) (*pubsub.PubSub, error) {
	optsPS := []pubsub.Option{
		pubsub.WithMessageSigning(true),
	}

	return pubsub.NewGossipSub(context.Background(), h, optsPS...)
}

// connectHostToPeer is used for connecting a host to another peer
func connectHostToPeer(h core.Host, connectToAddress string) {
	// Creating multi address
	multiAddr, err := multiaddr.NewMultiaddr(connectToAddress)
	if err != nil {
		fmt.Printf("Error : %v\n", err)
		return
	}

	pInfo, err := peer.AddrInfoFromP2pAddr(multiAddr)
	if err != nil {
		fmt.Printf("Error : %v\n", err)
		return
	}

	err = h.Connect(context.Background(), *pInfo)
	if err != nil {
		fmt.Printf("Error : %v\n", err)
	}
}
