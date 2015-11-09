// +build heavy

package integration

import (
	"bytes"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"io/ioutil"
	"log"
	"path/filepath"
	"testing"
	"time"
)

func TestSimpleSsh(t *testing.T) {
	Convey("Given we have a fresh git repository", t, func(c C) {
		ctx := newContext()
		// Initialize the git path
		log.Print("Reinitializing git")
		ctx.initializeGitRepo(c)

		// This config already has the data store set up properly
		conf := ctx.newWorkingConfig()

		Convey("When the Linux VM is running", func(c C) {

			// Set up one job every 3 seconds
			timeout := 20
			conf.JobStore.JobStoreLocal.JobGenerics = map[string]*config.Job{
				"linux_vm_job": &config.Job{
					CommandGeneric: &config.JobCommand{
						Expect:    []string{"vagrant@linux-vm:"},
						ExpectNot: []string{"No such file or directory"},
						Timeout:   &timeout,
					},
				},
			}
			conf.JobStore.JobStoreLocal.Jobs = map[string]*config.Job{
				"show_config": &config.Job{
					Generic:     "linux_vm_job",
					JobSchedule: &config.JobSchedule{Cron: "*/3 * * * * * *"},
					Commands: []*config.JobCommand{
						&config.JobCommand{Command: "cat /vagrant/sample-config.txt"},
					},
					// Let's go ahead and strip the "authenticated" from
					// "multilink bundle-name authenticated" with a replacer
					Scrubbers: []*config.JobScrubber{
						&config.JobScrubber{
							Type:    "simple",
							Search:  "multilink bundle-name authenticated",
							Replace: "multilink bundle-name {{replace_authenticated}}",
						},
					},
					TemplateValues: map[string]string{
						"replace_authenticated": "job-level",
					},
				},
			}

			// For the device itself
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
						"show_config": &config.Job{
							// Change the replace_authenticated template value
							TemplateValues: map[string]string{
								"replace_authenticated": "device-level",
							},
						},
					},
				},
			}

			Convey("And the controller and worker are started for 10 seconds to perform the backup", func(c C) {
				// Fire up the controller and worker
				controller := ctx.startControllerInBackground(c, conf)
				worker := ctx.startWorkerInBackground(c)

				log.Print("Waiting 10 seconds and then shutting down controller and worker")
				time.Sleep(time.Duration(10) * time.Second)
				So(worker.Stop(), ShouldBeNil)
				So(controller.Stop(), ShouldBeNil)
				time.Sleep(time.Duration(1) * time.Second)
				So(controller.Exited(), ShouldBeTrue)
				So(worker.Exited(), ShouldBeTrue)

				Convey("Then the git commit should be accurate", func(c C) {
					file, err := ioutil.ReadFile(filepath.Join(baseDirectory, "integration",
						"emulated", "sample-config.txt"))
					So(err, ShouldBeNil)
					// Expect the scrubbing and template value replacement
					file = bytes.Replace(file, []byte("multilink bundle-name authenticated"),
						[]byte("multilink bundle-name device-level"), -1)
					assertion := &gitAssertion{
						job:          "show_config",
						device:       "local_linux_vm",
						filesUpdated: []string{"by_device/local_linux_vm/show_config"},
						fileContents: string(file),
					}
					assertion.assertValid(ctx)
				})
			})
		})
	})
}
