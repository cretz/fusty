package controller

import (
	"errors"
	"fmt"
	"github.com/hashicorp/go-syslog"
	"gitlab.com/cretz/fusty/config"
	"log"
	"net/http"
	"os"
	"strconv"
)

// TODO: maybe remove this as a global var
var Verbose bool

type Controller struct {
	conf   *config.Config
	errLog *log.Logger
	outLog *log.Logger
	JobStore
	DeviceStore
	DataStore
	Scheduler
	started bool
}

// configFileName can be empty which means default config
func RunController(configFilename string) error {
	var conf *config.Config
	if configFilename == "" {
		if _, err := os.Stat("./fusty.conf.json"); err == nil {
			configFilename = "./fusty.conf.json"
		} else if _, err := os.Stat("./fusty.conf.yaml"); err == nil {
			configFilename = "./fusty.conf.yaml"
		} else if _, err := os.Stat("./fusty.conf.toml"); err == nil {
			configFilename = "./fusty.conf.toml"
		}
	}
	if configFilename != "" {
		if _, err := os.Stat(configFilename); os.IsNotExist(err) {
			return fmt.Errorf("Cannot find config file: %v", configFilename)
		} else if c, err := config.NewFromFile(configFilename); err != nil {
			return fmt.Errorf("Unable to read config file %v: %v", configFilename, err)
		} else {
			conf = c
		}
	} else {
		conf = config.NewDefault()
	}
	if Verbose {
		log.Printf("Creating controller with configuration: %v", conf)
	}
	cont, err := NewController(conf)
	if err != nil {
		return fmt.Errorf("Unable to start controller: %v", err)
	}
	if err := cont.Start(); err != nil {
		return fmt.Errorf("Unable to start controller: %v", err)
	}
	return nil
}

func NewController(conf *config.Config) (*Controller, error) {
	controller := &Controller{conf: conf}
	if conf.Syslog {
		if logger, err := gsyslog.NewLogger(gsyslog.LOG_ERR, "LOCAL0", "fusty"); err != nil {
			return nil, fmt.Errorf("Unable to create syslog: %v", err)
		} else {
			controller.errLog = log.New(logger, "", log.LstdFlags)
		}
		if logger, err := gsyslog.NewLogger(gsyslog.LOG_INFO, "LOCAL0", "fusty"); err != nil {
			return nil, fmt.Errorf("Unable to create syslog: %v", err)
		} else {
			controller.outLog = log.New(logger, "", log.LstdFlags)
		}
	} else {
		controller.errLog = log.New(os.Stderr, "", log.LstdFlags)
		controller.outLog = log.New(os.Stdout, "", log.LstdFlags)
	}
	if conf.JobStore == nil {
		return nil, errors.New("Job store configuration not found")
	} else if jobStore, err := NewJobStoreFromConfig(conf.JobStore); err != nil {
		return nil, fmt.Errorf("Unable to create job store: %v", err)
	} else {
		controller.JobStore = jobStore
	}
	if conf.DeviceStore == nil {
		return nil, errors.New("Device store configuration not found")
	} else if deviceStore, err := NewDeviceStoreFromConfig(conf.DeviceStore, controller.JobStore); err != nil {
		return nil, fmt.Errorf("Unable to create device store: %v", err)
	} else {
		controller.DeviceStore = deviceStore
	}
	if conf.DataStore == nil {
		return nil, errors.New("Data store configuration not found")
	} else if dataStore, err := NewDataStoreFromConfig(conf.DataStore); err != nil {
		return nil, fmt.Errorf("Unable to create data store: %v", err)
	} else {
		controller.DataStore = dataStore
	}
	if scheduler, err := controller.NewLocalScheduler(); err != nil {
		return nil, fmt.Errorf("Unable to create scheduler: %v", err)
	} else {
		controller.Scheduler = scheduler
	}
	return controller, nil
}

func (c *Controller) Start() error {
	if c.started {
		return errors.New("Controller already started")
	}
	ip := c.conf.Ip
	if ip == "" {
		ip = "0.0.0.0"
	}
	port := c.conf.Port
	if port == 0 {
		port = 9400
	}
	mux := http.NewServeMux()
	c.addApiHandlers(mux)
	server := &http.Server{
		Addr:    ip + ":" + strconv.Itoa(port),
		Handler: mux,
	}
	// TODO: TLS support
	c.outLog.Printf("Starting controller on %v", server.Addr)
	return server.ListenAndServe()
}
