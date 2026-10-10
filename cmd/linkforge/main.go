// Command linkforge is the Linkforge service binary.
package main

import (
	"fmt"

	"github.com/avekrivoy/linkforge/internal/base62"
)

// version is overwritten at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	fmt.Printf("linkforged %s\n", version)
	fmt.Println(base62.Encode(123456))
	fmt.Printf("linkforge %s\n", version)
}
