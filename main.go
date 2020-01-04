package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p-core/crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"

	"github.com/multiformats/go-multiaddr"
)

func main() {
	fmt.Println("startup")

	sourcePort := 5555

	r := rand.New(rand.NewSource(int64(sourcePort)))

	// Creates a new RSA key pair for this host.
	prvKey, _, err := crypto.GenerateKeyPairWithReader(crypto.RSA, 2048, r)
	if err != nil {
		panic(err)
	}

	sourceMultiAddr, _ := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/0.0.0.0/tcp/%d", sourcePort))

	host, err := libp2p.New(
		context.Background(),
		libp2p.ListenAddrs(sourceMultiAddr),
		libp2p.Identity(prvKey),
	)
	if err != nil {
		panic(err)
	}

	router, err := pubsub.NewGossipSub(context.Background(), host)
	if err != nil {
		panic(err)
	}

	for _, peer := range router.ListPeers("") {
		fmt.Println(peer.ShortString())
	}

	fmt.Printf("Hello World, my hosts ID is %s\n", host.ID())

}
