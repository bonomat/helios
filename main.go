package main

import (
	"context"
	"fmt"

	"github.com/libp2p/go-libp2p"
)

func main() {
	fmt.Println("startup")

	// The context governs the lifetime of the libp2p node
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// To construct a simple host with all the default settings, just use `New`
	h, err := libp2p.New(ctx)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Hello World, my hosts ID is %s\n", h.ID())
}
