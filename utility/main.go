package main

import (
	"fmt"
	"github.com/tliron/puccini/utility/clout"
	"os"
)

func main() {
	args := os.Args
	if len(args) < 3 {
		fmt.Println("Usage: go run main.go file1 file2")
		os.Exit(1)
	}
	clout.Compare(args[1], args[2])
}
