package main

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"flag"
	"fmt"

	"os"
	"sync"
	"time"

	"github.com/ipfs/go-log"
	logging "github.com/whyrusleeping/go-logging"

	"github.com/btcsuite/btcd/btcec"
	"github.com/libp2p/go-libp2p"
	core "github.com/libp2p/go-libp2p-core"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/protocol"
	crypto "github.com/libp2p/go-libp2p-crypto"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/multiformats/go-multiaddr"
)

var logger = log.Logger("helios")

func startListening(ctx context.Context, ps libp2pPubSub, host core.Host) {
	wg := &sync.WaitGroup{}

	wg.Add(1)
	go func(host *core.Host, pubSub *libp2pPubSub) {
		for {
			msg, err := pubSub.subscription.Next(ctx)
			if err != nil {
				logger.Errorf("Error: %v\n", err)
				continue
			}

			if (*host).ID() != msg.ReceivedFrom {
				stream, err := (*host).NewStream(ctx, msg.ReceivedFrom, protocol.ID("/chat/1.1.0"))
				if err != nil {
					logger.Errorf("Error: %v\n", err)
					continue
				}
				_, err = stream.Write([]byte("test"))
				if err != nil {
					logger.Errorf("Error: %v\n", err)
					continue
				}

				rw := bufio.NewReadWriter(bufio.NewReader(stream), bufio.NewWriter(stream))

				go readData(rw)
			}

			logger.Infof("Node %s received Message: '%s' from '%s' \n", (*host).ID().Pretty(), string(msg.Data), string(msg.ReceivedFrom.Pretty()))
		}
	}(&host, &ps)

	logger.Infof("Broadcasting a message ...")
	err := ps.pubsub.Publish("TOPIC", []byte("Bananaboatckr"))
	if err != nil {
		logger.Errorf("Error: %v\n", err)
	}

	wg.Wait()
	logger.Infof("The END")
}

func main() {
	log.SetAllLoggers(logging.WARNING)
	log.SetLogLevel("helios", "info")

	help := flag.Bool("h", false, "Display Help")
	config, err := ParseFlags()
	if err != nil {
		panic(err)
	}

	if *help {
		logger.Infof("Usage: Run go run github.com/bonomat/helios in different terminals.")
		flag.PrintDefaults()
		return
	}

	ctx := context.Background()

	host, err := createHost(ctx, config.ListenAddresses)
	if err != nil {
		panic(err)
	}

	logger.Infof(config.ListenAddresses[0].String() + "/p2p/" + host.ID().String())

	ps := new(libp2pPubSub)
	// creating pubsubs
	err = ps.initializePubSub(ctx, host)
	if err != nil {
		panic(err)
	}

	defer func() {
		logger.Info("Closing host")
		host.Close()
		ps.subscription.Cancel()
	}()

	for _, peer := range config.BootstrapPeers {
		connectHostToPeer(ctx, host, peer)
	}

	startListening(ctx, *ps, host)
}

type libp2pPubSub struct {
	pubsub       *pubsub.PubSub       // PubSub of each individual node
	subscription *pubsub.Subscription // Subscription of individual node
	topic        string               // PubSub topic
}

// initializePubSub creates a PubSub for the peer and also subscribes to a topic
func (c *libp2pPubSub) initializePubSub(ctx context.Context, host core.Host) (err error) {
	optsPS := []pubsub.Option{
		pubsub.WithMessageSigning(true),
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
		logger.Errorf("Error: %v\n", err)
		return
	}

	err = h.Connect(ctx, *pInfo)
	if err != nil {
		logger.Errorf("Error: %v\n", err)
	}

	// wait for subscription to be propagated
	time.Sleep(time.Second * 2)
}

func readData(rw *bufio.ReadWriter) {
	for {
		str, err := rw.ReadString('\n')
		if err != nil {
			logger.Errorf("Error: %v\n", err)
			panic(err)
		}

		if str == "" {
			return
		}
		if str != "\n" {
			// Green console colour: 	\x1b[32m
			// Reset console colour: 	\x1b[0m
			logger.Infof("\x1b[32m%s\x1b[0m> ", str)
		}

	}
}

func writeData(rw *bufio.ReadWriter) {
	stdReader := bufio.NewReader(os.Stdin)

	for {
		logger.Info("> ")
		sendData, err := stdReader.ReadString('\n')
		if err != nil {
			logger.Errorf("Error reading from stdin: %v\n", err)
			panic(err)
		}

		_, err = rw.WriteString(fmt.Sprintf("%s\n", sendData))
		if err != nil {
			logger.Errorf("Error writing to buffer: %v\n", err)
			panic(err)
		}
		err = rw.Flush()
		if err != nil {
			logger.Errorf("Error flushing buffer: %v\n", err)
			panic(err)
		}
	}
}
