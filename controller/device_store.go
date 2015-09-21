package controller

import (
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
	"gitlab.com/cretz/fusty/model"
	"log"
)

type DeviceStore interface {
	AllDevices() map[string]*model.Device
}

func NewDeviceStoreFromConfig(conf *config.DeviceStore, jobStore JobStore) (DeviceStore, error) {
	switch conf.Type {
	case "local":
		return newLocalDeviceStore(conf.DeviceStoreLocal, jobStore)
	default:
		return nil, fmt.Errorf("Unrecognized device store type: %v", conf.Type)
	}
}

type localDeviceStore struct {
	devices map[string]*model.Device
}

func newLocalDeviceStore(conf *config.DeviceStoreLocal, jobStore JobStore) (*localDeviceStore, error) {
	if Verbose {
		log.Printf("Loading devices from config")
	}
	store := &localDeviceStore{devices: make(map[string]*model.Device)}
	errs := []error{}
	for name, confDevice := range conf.Devices {
		device := model.NewDefaultDevice(name)
		// Add all of the jobs
		device.Jobs = make(map[string]*model.Job)
		for name, _ := range confDevice.Jobs {
			job := jobStore.AllJobs()[name]
			if job == nil {
				job = model.NewDefaultJob(name)
			} else {
				job = job.DeepCopy()
			}
			device.Jobs[name] = job
		}
		// Generic first if present
		if confDevice.Generic != "" {
			generic := conf.DeviceGenerics[confDevice.Generic]
			if generic == nil {
				errs = append(errs, fmt.Errorf("Unable to find device generic named: %v", confDevice.Generic))
				continue
			}
			if err := device.ApplyConfig(generic); err != nil {
				errs = append(errs, fmt.Errorf("Error applying device generic %v: %v", confDevice.Generic, err))
				continue
			}
		} else if generic := conf.DeviceGenerics["default"]; generic != nil {
			if err := device.ApplyConfig(generic); err != nil {
				errs = append(errs, fmt.Errorf("Error applying default device generic: %v", err))
				continue
			}
		}
		// Specific device settings
		if err := device.ApplyConfig(confDevice); err != nil {
			errs = append(errs, fmt.Errorf("Error configuring device %v: %v", device.Name, err))
			continue
		}
		// Validate the device
		if _, ok := store.devices[device.Name]; ok {
			errs = append(errs, fmt.Errorf("Ambiguous device name %v", device.Name))
			continue
		}
		if validationErrors := device.Validate(); len(validationErrors) > 0 {
			errs = append(errs, validationErrors...)
			continue
		}
		store.devices[device.Name] = device
	}
	// Any errors, combine into single error
	if len(errs) > 0 {
		msg := "Device validation failed:"
		for _, err := range errs {
			msg += "\n" + err.Error()
		}
		return nil, errors.New(msg)
	}
	return store, nil
}

func (l *localDeviceStore) AllDevices() map[string]*model.Device {
	// We trust callers not to modify this
	return l.devices
}
