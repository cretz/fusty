package controller

import (
	"gitlab.com/cretz/fusty/controller/config"
	"log"
)

type Controller struct {
	*config.Config
	DeviceStore
	Scheduler
	*log.Logger
}

type DeviceStore interface {
	AllDevices() map[string]*Device
}
