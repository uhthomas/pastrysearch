package main

import (
	"context"
	"crypto/rand"
	"flag"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/uhthomas/pastry/pkg/pastry"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/sync/errgroup"
)

func main() {
	addr := flag.String("addr", ":2376", "The address to listen on")
	dial := flag.String("dial", "", "a comma separated list of addresses to connect to")
	flag.Parse()

	// Generate key for node
	var seed [ed25519.SeedSize]byte
	if _, err := io.ReadFull(rand.Reader, seed[:]); err != nil {
		log.Fatal(err)
	}

	n, err := pastry.New(
		pastry.Seed(seed[:]),
		pastry.Forward(pastry.ForwarderFunc(func(key, b, next []byte) {
			// we need to intercept these messages in a sort of "fan-out" style due to the way the bloom
			// filters work
		})),
		pastry.Deliver(pastry.DelivererFunc(func(key, b []byte) {
			// we need to either store the record or connect to the peer and tell them what records we have
			// which match their request
		})),
	)
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error { return n.ListenAndServe(ctx, "tcp", *addr) })

	g.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)

		select {
		case <-c:
			cancel()
			return n.Close()
		case <-ctx.Done():
			return ctx.Err()
		}
	})

	log.Printf("Listening on %s\n", *addr)

	if s := strings.Fields(strings.TrimSpace(*dial)); len(s) > 0 {
		log.Printf("Connecting to %d nodes\n", len(s))
		for _, addr := range s {
			log.Printf("Connecting to %s\n", addr)
			conn, err := net.Dial("tcp", addr)
			if err != nil {
				log.Fatal(err)
			}
			go n.Accept(conn)
		}
	}

	if err := g.Wait(); err != nil {
		log.Fatal(err)
	}
}
