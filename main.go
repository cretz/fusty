package main

import (
	"errors"
	"flag"
	"fmt"
	"gitlab.com/cretz/fusty/controller"
	"log"
	"os"
)

func main() {
	if len(os.Args) <= 1 {
		log.Fatal("Command required")
	}
	if err := run(os.Args[1], os.Args[2:]...); err != nil {
		log.Fatal(err)
	}
}

func run(command string, args ...string) error {
	switch command {
	case "controller":
		return runController(args...)
	case "worker":
		return runWorker(args...)
	case "help":
		return runHelp(args...)
	default:
		return fmt.Errorf("Unrecognized command: %v", command)
	}
}

func runController(args ...string) error {
	flags := flag.NewFlagSet("flags", flag.ContinueOnError)
	configFile := flags.String("config", "", "Configuration file")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("Error parsing arguments: %v", err)
	} else if flags.NArg() != 0 {
		return errors.New("Controller only accepts single config-file argument at most")
	}
	return controller.RunController(*configFile)
}

func runWorker(args ...string) error {
	return errors.New("TODO")
}

func runHelp(args ...string) error {
	return errors.New("TODO")
}
