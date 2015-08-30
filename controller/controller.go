package controller

import (
	"gitlab.com/cretz/fusty/config"
	"log"
)

type Controller struct {
	*config.Config
	DeviceStore
	Scheduler
	DataStore
	*log.Logger
}
