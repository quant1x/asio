package main

import (
	"flag"
	"fmt"
)

func main() {
	numLoops := 1
	flag.IntVar(&numLoops, "loops", 1, "num loops")
	flag.Parse()

	fmt.Println("其他参数: ", flag.Args())

	var a uint32 = (1 << 31)
	var b int32 = -0x80000000
	if a == uint32(b) {
		fmt.Println("==")
	} else {
		fmt.Println("!=")
	}

}
