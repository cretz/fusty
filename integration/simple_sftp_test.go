// +build heavy

package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"log"
	"testing"
	"time"
)

func TestSimpleSftp(t *testing.T) {
	Convey("Given we have a fresh git repository", t, func(c C) {
		ctx := newContext()
		// Initialize the git path
		log.Print("Reinitializing git")
		ctx.initializeGitRepo(c)

		// This config already has the data store set up properly
		conf := ctx.newWorkingConfig()

		Convey("When a Juniper emulated environment is running", func(c C) {
			juniper := newDefaultEmulatedJuniperDevice()
			juniper.assertOnline()

			// Set up one job every 3 seconds
			conf.JobStore.JobStoreLocal.JobGenerics = map[string]*config.Job{"juniper": juniper.genericJob()}
			conf.JobStore.JobStoreLocal.Jobs = map[string]*config.Job{
				"simple": &config.Job{
					Generic:     "juniper",
					JobSchedule: &config.JobSchedule{Cron: "*/3 * * * * * *"},
				},
			}

			// For the Juniper VM
			conf.DeviceStore.DeviceStoreLocal.DeviceGenerics =
				map[string]*config.Device{"juniper": juniper.genericDevice()}
			conf.DeviceStore.DeviceStoreLocal.Devices = map[string]*config.Device{
				"local": &config.Device{
					Generic: "juniper",
					Jobs: map[string]*config.Job{
						"simple": &config.Job{},
					},
				},
			}

			Convey("And the controller and worker are started for 5 seconds to perform the backup", func(c C) {
				// Fire up the controller and worker
				controller := ctx.startControllerInBackground(c, conf)
				worker := ctx.startWorkerInBackground(c)

				// Wait for 5 seconds and shut em down...
				log.Print("Waiting 5 seconds and then shutting down controller and worker")
				time.Sleep(time.Duration(5) * time.Second)
				So(worker.Stop(), ShouldBeNil)
				So(controller.Stop(), ShouldBeNil)
				time.Sleep(time.Duration(1) * time.Second)
				So(controller.Exited(), ShouldBeTrue)
				So(worker.Exited(), ShouldBeTrue)

				Convey("Then the git commit should be accurate", func(c C) {
					ctx.assertValidGitCommit("ge-0/0/0")
				})
			})
		})
	})
}
