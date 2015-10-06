// +build heavy2

package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSimpleSsh(t *testing.T) {
	Convey("Given we have a fresh git repository", t, func(c C) {
		// Initialize the git path
		log.Print("Reinitializing git")
		cleanAndReinitializeGitRepo(c)

		// This config already has the data store set up properly
		conf := newWorkingConfig()

		Convey("When an SSH emulated environment is running", func(c C) {
			log.Print("Starting a device")
			device := startDefaultDevice()
			Reset(device.stop)

			// Set up one job every 3 seconds
			conf.JobStore.JobStoreLocal.JobGenerics = map[string]*config.Job{"simplebase": defaultDeviceGenericJob()}
			conf.JobStore.JobStoreLocal.Jobs = map[string]*config.Job{
				"simple": &config.Job{
					Generic:     "sshbase",
					JobSchedule: &config.JobSchedule{Cron: "*/3 * * * * * *"},
				},
			}

			// For the device itself
			conf.DeviceStore.DeviceStoreLocal.DeviceGenerics =
				map[string]*config.Device{"localbase": device.genericDevice()}
			conf.DeviceStore.DeviceStoreLocal.Devices = map[string]*config.Device{
				"local": &config.Device{
					Generic: "localbase",
					Jobs: map[string]*config.Job{
						"simple": &config.Job{},
					},
				},
			}

			Convey("And the controller and worker are started for 5 seconds to perform the backup", func(c C) {
				// Fire up the controller and worker
				controller := startController(c, conf)
				worker := startWorker(c)

				// Wait for 5 seconds and shut em down...
				log.Print("Waiting 5 seconds and then shutting down controller and worker")
				time.Sleep(time.Duration(5) * time.Second)
				So(worker.Stop(), ShouldBeNil)
				So(controller.Stop(), ShouldBeNil)
				time.Sleep(time.Duration(1) * time.Second)
				So(controller.Exited(), ShouldBeTrue)
				So(worker.Exited(), ShouldBeTrue)

				Convey("Then the git commit should be accurate", func(c C) {
					assertValidGitCommit()
				})
			})
		})
	})
}

func startDefaultDevice() *mockDevice {
	device := &mockDevice{
		username: "johndoe",
		password: "secretpass",
		responses: map[string]string{
			"command1": strings.Repeat("This is a command1 response\n", 5),
			"command2": strings.Repeat("This is a command2 response\n", 5),
		},
	}
	So(device.listen(), ShouldBeNil)
	go device.acceptUntilError()
	return device
}

func defaultDeviceGenericJob() *config.Job {
	return &config.Job{
		Commands: []*config.JobCommand{
			&config.JobCommand{Command: "command1"},
			&config.JobCommand{Command: "command2"},
		},
	}
}
