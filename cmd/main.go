package main

import (
	"flag"
	"fmt"
	"os"
)

// version is set during build using -X linker flag
var version = "dev"

func main() {
	// Parse command line flags
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	// Show version if requested
	if *showVersion {
		fmt.Printf("winterflow-agent version %s\n", version)
		os.Exit(0)
	}

	// TODO: Add your agent initialization and main logic here
	fmt.Println("Winterflow Agent starting...")
}
