package main

import (
	"errors"
	"flag"
	"fmt"
	"gitlab.com/cretz/fusty/controller"
	"gitlab.com/cretz/fusty/worker"
	"log"
	"os"
	"strings"
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
	flags := flag.NewFlagSet("flags", flag.ContinueOnError)
	conf := &worker.Config{}
	flags.StringVar(&conf.ControllerUrl, "controller", "", "Base URL for controller")
	var tags multistring
	flags.Var(&tags, "tag", "One or more tags")
	flags.IntVar(&conf.SleepSeconds, "sleep", 15, "Sleep seconds")
	flags.IntVar(&conf.MaxJobs, "maxjobs", 2000, "Max running jobs")
	flags.IntVar(&conf.TimeoutSeconds, "timeout", 3, "Controller HTTP timeout seconds")
	if err := flags.Parse(args); err != nil {
		return fmt.Errorf("Error parsing arguments: %v", err)
	} else if flags.NArg() != 0 {
		return fmt.Errorf("Unrecognized extra parameter: %v", flags.Arg(0))
	}
	conf.Tags = tags
	return worker.RunWorker(conf)
}

func runHelp(args ...string) error {
	return errors.New("TODO")
}

type multistring []string
func (m *multistring) Set(value string) error {
	*m = append(*m, value)
	return nil
}
func (m *multistring) String() string {
	return strings.Join(*m, ", ")
}
