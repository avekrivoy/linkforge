// Command linkforge is the Linkforge service binary.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/avekrivoy/linkforge/internal/link"
	"github.com/avekrivoy/linkforge/internal/store"
)

// version is overwritten at build time with -ldflags "-X main.version=...".
// var version = "dev"

func main() {
	ctx := context.Background()
	args := os.Args[1:]
	storage := store.New()
	svc := link.NewService(storage)
	ownerID := "demo"

	for _, l := range args {

		req := link.ShortenRequest{
			Target:  l,
			OwnerID: ownerID,
		}

		_, err := svc.Shorten(ctx, req)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			continue
		}
	}

	links, err := svc.List(ctx, ownerID, link.Page{})

	if err != nil {
		fmt.Printf("couldn't list %s %s", ownerID, err)
	}

	for _, v := range links {
		fmt.Printf("%v\t%v\n", v.Code, v.Target)
	}

	fmt.Printf("Total: %v\n", storage.Count())
}
