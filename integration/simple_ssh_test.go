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
						"show_config": &config.Job{},
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

					assertion := &gitAssertion{
						job:          "show_config",
						device:       "local_linux_vm",
						filesUpdated: []string{"by_device/local_linux_vm/show_config"},
						fileContents: string(file),
					}
					assertion.assertValid(ctx)
					// Gotta compare regardless of newline style
					//					assertValidGitCommit(ctx, strings.Replace(strings.TrimSpace(string(file)), "\r\n", "\n", -1))
				})
			})
		})
	})
}

//func assertValidGitCommit(ctx *context, fileContentSubstring string) {
//	gitAssertDir, err := ioutil.TempDir(ctx.tempDirectory, "git-assert-temp")
//	So(err, ShouldBeNil)
//	So(os.MkdirAll(gitAssertDir, os.ModePerm), ShouldBeNil)
//	runInDir(gitAssertDir, "git", "clone", ctx.gitRepoDirectory, gitAssertDir)
//
//	authorName := runInDir(gitAssertDir, "git", "log", "-1", "--pretty=%an")
//	So(authorName, ShouldEqual, "John Doe")
//	authorEmail := runInDir(gitAssertDir, "git", "log", "-1", "--pretty=%ae")
//	So(authorEmail, ShouldEqual, "jdoe@example.com")
//	commitComment := runInDir(gitAssertDir, "git", "log", "-1", "--pretty=%B")
//	So(commitComment, ShouldContainSubstring, "Job: show_config\n")
//	So(commitComment, ShouldContainSubstring, "Device: local_linux_vm\n")
//	// TODO: Some extra validation of the values here?
//	So(commitComment, ShouldContainSubstring, "Expected Run Date:")
//	So(commitComment, ShouldContainSubstring, "Start Date:")
//	So(commitComment, ShouldContainSubstring, "End On:")
//	So(commitComment, ShouldContainSubstring, "Elapsed Time:")
//	filesText := runInDir(gitAssertDir, "git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
//	filesUpdated := strings.Split(filesText, "\n")
//	// TODO: Fix this when checking for other types of git structures
//	So(len(filesUpdated), ShouldEqual, 1)
//	So(filesUpdated, ShouldContain, "by_device/local_linux_vm/show_config")
//
//	// Now read the the file and make sure it looks right
//	fileBytes, err := ioutil.ReadFile(filepath.Join(gitAssertDir, "by_device/local_linux_vm/show_config"))
//	So(err, ShouldBeNil)
//	// Change /r/n to /n
//	fileString := strings.Replace(string(fileBytes), "\r\n", "\n", -1)
//	So(fileString, ShouldContainSubstring, fileContentSubstring)
//}
