package main

import (
	"fmt"
	"time"

	"github.com/felixge/pager"
)

func main() {
	pager, err := pager.Start("less")
	if err != nil {
		panic(err)
	}
	for l := 0; l < 1000; l++ {
		fmt.Printf("Line %d\n", l)
		time.Sleep(time.Second / 2000)
	}
	if err := pager.Wait(); err != nil {
		panic(err)
	}
	fmt.Printf("DONE\n")
}
