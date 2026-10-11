// Command linkforge is the Linkforge service binary.
package main

import (
	"fmt"
	"os"

	"github.com/avekrivoy/linkforge/internal/link"
	"github.com/avekrivoy/linkforge/internal/store"
)

// version is overwritten at build time with -ldflags "-X main.version=...".
// var version = "dev"

func main() {
	links := os.Args[1:]

	s := store.New()

	for pos, lnk := range links {
		l, err := link.New(uint64(pos+1), lnk)

		if err != nil {
			panic(err)
		}
		err = store.Add(s, l)

		if err != nil {
			panic(err)
		}
		fmt.Printf("%v\t%s\n", l.Code, l.Target)
	}

	fmt.Printf("Total: %v\n", store.Count(s))
}
