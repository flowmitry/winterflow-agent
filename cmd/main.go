package main

import (
	"flag"
	"fmt"
	"os"
	"winterflow-agent/pkg/agent"
)

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("WinterFlow Agent version: %s (#%d)\n", agent.GetVersion(), agent.GetNumericVersion())
		os.Exit(0)
	}
}
