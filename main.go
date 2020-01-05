package main

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"flag"
	"fmt"
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

func startListening(ps libp2pPubSub, host core.Host) {
	wg := &sync.WaitGroup{}

	ctx := context.Background()

	wg.Add(1)
	go func(host *core.Host, pubSub *libp2pPubSub) {
		for true {
			msg, err := pubSub.subscription.Next(ctx)
			if err != nil {
				fmt.Println(err.Error())
			}

			fmt.Printf("Node %s received Message: '%s' from '%s' \n", (*host).ID().Pretty(), string(msg.Data), string(msg.ReceivedFrom.Pretty()))
		}
	}(&host, &ps)

	fmt.Println("Broadcasting a message ...")
	err := ps.pubsub.Publish("TOPIC", []byte("Bananaboatckr"))
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	wg.Wait()
	fmt.Println("The END")
}

func main() {
	help := flag.Bool("h", false, "Display Help")
	config, err := ParseFlags()
	if err != nil {
		panic(err)
	}

	if *help {
		fmt.Println("Usage: Run go run github.com/bonomat/helios in different terminals.")
		flag.PrintDefaults()
		return
	}

	ctx := context.Background()

	host, err := createHost(ctx, config.ListenAddresses)
	if err != nil {
		panic(err)
	}

	fmt.Println(config.ListenAddresses[0].String() + "/p2p/" + host.ID().String())

	ps := new(libp2pPubSub)
	// creating pubsubs
	err = ps.initializePubSub(ctx, host)
	if err != nil {
		panic(err)
	}

	defer func() {
		fmt.Println("Closing host")
		host.Close()
	}()

	for _, peer := range config.BootstrapPeers {
		connectHostToPeer(ctx, host, peer)
		// wait for network to be propagated
		time.Sleep(time.Second * 2)
	}

	startListening(*ps, host)
}

type libp2pPubSub struct {
	pubsub       *pubsub.PubSub       // PubSub of each individual node
	subscription *pubsub.Subscription // Subscription of individual node
	topic        string               // PubSub topic
}

// initializePubSub creates a PubSub for the peer and also subscribes to a topic
func (c *libp2pPubSub) initializePubSub(ctx context.Context, host core.Host) (err error) {
	optsPS := []pubsub.Option{
		pubsub.WithMessageSigning(false),
	}

	c.pubsub, err = pubsub.NewGossipSub(ctx, host, optsPS...)
	if err != nil {
		return err
	}

	// Registering to the topic
	c.topic = "TOPIC"

	topic, err := c.pubsub.Join(c.topic)
	if err != nil {
		return err
	}

	// Creating a subscription and subscribing to the topic
	c.subscription, err = topic.Subscribe()
	if err != nil {
		return err
	}

	return nil
}

// createHost creates a host with some default options and a signing identity
func createHost(ctx context.Context, addresses addrList) (host core.Host, err error) {
	// Producing private key
	prvKey, err := ecdsa.GenerateKey(btcec.S256(), rand.Reader)
	if err != nil {
		return host, err
	}

	sk := (*crypto.Secp256k1PrivateKey)(prvKey)

	// Starting a peer with default configs
	opts := []libp2p.Option{
		libp2p.ListenAddrs([]multiaddr.Multiaddr(addresses)...),
		libp2p.Identity(sk),
		libp2p.DefaultTransports,
		libp2p.DefaultMuxers,
		libp2p.DefaultSecurity,
	}

	return libp2p.New(ctx, opts...)
}

// connectHostToPeer is used for connecting a host to another peer
func connectHostToPeer(ctx context.Context, h core.Host, address multiaddr.Multiaddr) {
	pInfo, err := peer.AddrInfoFromP2pAddr(address)
	if err != nil {
		fmt.Printf("Error : %v\n", err)
		return
	}

	err = h.Connect(ctx, *pInfo)
	if err != nil {
		fmt.Printf("Error : %v\n", err)
	}
}
