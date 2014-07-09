package main

import (
	"fmt"

	"github.com/felixge/pager"
)

func main() {
	pager, err := pager.Start("less")
	if err != nil {
		panic(err)
	}
	for l := 0; l < 100000; l++ {
		fmt.Printf("Line %d\n", l)
	}
	if err := pager.Wait(); err != nil {
		panic(err)
	}
	fmt.Printf("DONE\n")
}
