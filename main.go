package main

import (
	"os"

	"github.com/marsvillager/simbeat/cmd"

	_ "github.com/marsvillager/simbeat/include"
)

func main() {
	if err := cmd.RootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
