// +build heavy

package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"io/ioutil"
	"log"
	"path/filepath"
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

		Convey("When the Linux VM is running", func(c C) {

			// Set up one job every 3 seconds
			conf.JobStore.JobStoreLocal.JobGenerics = map[string]*config.Job{
				"linux_file_job": &config.Job{
					Type: "file",
					JobFile: map[string]*config.JobFile{
						"/vagrant/sample-config.txt.gz": &config.JobFile{Compression: "gzip"},
					},
				},
			}
			conf.JobStore.JobStoreLocal.Jobs = map[string]*config.Job{
				"get_file": &config.Job{
					Generic:     "linux_file_job",
					JobSchedule: &config.JobSchedule{Cron: "*/3 * * * * * *"},
				},
			}

			// For the device
			conf.DeviceStore.DeviceStoreLocal.DeviceGenerics = map[string]*config.Device{
				"linux_vm_base": &config.Device{
					Host: "127.0.0.1",
					DeviceProtocol: &config.DeviceProtocol{
						Type:              "ssh",
						DeviceProtocolSsh: &config.DeviceProtocolSsh{Port: 3222},
					},
					DeviceCredentials: &config.DeviceCredentials{User: "vagrant", Pass: "vagrant"},
				},
			}
			conf.DeviceStore.DeviceStoreLocal.Devices = map[string]*config.Device{
				"local_linux_vm": &config.Device{
					Generic: "linux_vm_base",
					Jobs: map[string]*config.Job{
						"get_file": &config.Job{},
					},
				},
			}

			Convey("And the controller and worker are started for 10 seconds to perform the backup", func(c C) {
				// Fire up the controller and worker
				controller := ctx.startControllerInBackground(c, conf, true)
				worker := ctx.startWorkerInBackground(c)

				log.Print("Waiting 10 seconds and then shutting down controller and worker")
				time.Sleep(time.Duration(10) * time.Second)
				So(worker.Stop(), ShouldBeNil)
				So(controller.Stop(), ShouldBeNil)
				time.Sleep(time.Duration(1) * time.Second)
				So(controller.Exited(), ShouldBeTrue)
				So(worker.Exited(), ShouldBeTrue)

				Convey("Then the git commit should be accurate", func(c C) {
					//ctx.assertValidGitCommit("ge-0/0/0")
					file, err := ioutil.ReadFile(filepath.Join(baseDirectory, "integration",
						"emulated", "sample-config.txt"))
					So(err, ShouldBeNil)
					assertion := &gitAssertion{
						job:          "get_file",
						device:       "local_linux_vm",
						filesUpdated: []string{"by_device/local_linux_vm/get_file"},
						fileContents: string(file),
					}
					assertion.assertValid(ctx)
				})
			})
		})
	})
}
