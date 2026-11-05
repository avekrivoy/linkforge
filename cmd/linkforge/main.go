// Command linkforge is the Linkforge service binary.
package main

// version is overwritten at build time with -ldflags "-X main.version=...".
// var version = "dev"

func main() {
	// var id uint64

	// links := os.Args[1:]
	// storage := store.New()

	// for _, l := range links {
	// 	lnk, err := link.New(id+1, l)

	// 	if err != nil {
	// 		fmt.Fprintln(os.Stderr, err)
	// 		continue
	// 	}
	// 	err = store.Add(storage, lnk)

	// 	if err != nil {
	// 		fmt.Fprintln(os.Stderr, err)
	// 		continue
	// 	}
	// 	id += 1

	// 	fmt.Printf("%v\t%s\n", lnk.Code, lnk.Target)
	// }

	// fmt.Printf("Total: %v\n", store.Count(storage))
}
