// +build heavy

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

func TestSimpleSftp(t *testing.T) {
	Convey("Given we have a fresh git repository", t, func(c C) {
		// Initialize the git path
		log.Print("Reinitializing git")
		cleanAndReinitializeGitRepo(c)

		// This config already has the data store set up properly
		conf := newWorkingConfig()

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
				controller := startControllerInBackground(c, conf)
				worker := startWorkerInBackground(c)

				// Wait for 5 seconds and shut em down...
				log.Print("Waiting 5 seconds and then shutting down controller and worker")
				time.Sleep(time.Duration(5) * time.Second)
				So(worker.Stop(), ShouldBeNil)
				So(controller.Stop(), ShouldBeNil)
				time.Sleep(time.Duration(1) * time.Second)
				So(controller.Exited(), ShouldBeTrue)
				So(worker.Exited(), ShouldBeTrue)

				Convey("Then the git commit should be accurate", func(c C) {
					assertValidSftpGitCommit()
				})
			})
		})
	})
}

func assertValidSftpGitCommit() {
	gitAssertDir, err := ioutil.TempDir(tempDirectory, "git-assert-temp")
	So(err, ShouldBeNil)
	So(os.MkdirAll(gitAssertDir, os.ModePerm), ShouldBeNil)
	runInDir(gitAssertDir, "git", "clone", gitRepoDirectory, gitAssertDir)

	authorName := runInDir(gitAssertDir, "git", "log", "-1", "--pretty=%an")
	So(authorName, ShouldEqual, "John Doe")
	authorEmail := runInDir(gitAssertDir, "git", "log", "-1", "--pretty=%ae")
	So(authorEmail, ShouldEqual, "jdoe@example.com")
	commitComment := runInDir(gitAssertDir, "git", "log", "-1", "--pretty=%B")
	So(commitComment, ShouldContainSubstring, "Job: simple\n")
	So(commitComment, ShouldContainSubstring, "Device: local\n")
	// TODO: Some extra validation of the values here?
	So(commitComment, ShouldContainSubstring, "Expected Run Date:")
	So(commitComment, ShouldContainSubstring, "Start Date:")
	So(commitComment, ShouldContainSubstring, "End On:")
	So(commitComment, ShouldContainSubstring, "Elapsed Time:")
	filesText := runInDir(gitAssertDir, "git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
	filesUpdated := strings.Split(filesText, "\n")
	// TODO: Fix this when checking for other types of git structures
	So(len(filesUpdated), ShouldEqual, 1)
	So(filesUpdated, ShouldContain, "by_device/local/simple")

	// Now read the the file and make sure it looks right
	fileBytes, err := ioutil.ReadFile(filepath.Join(gitAssertDir, "by_device/local/simple"))
	So(err, ShouldBeNil)
	So(string(fileBytes), ShouldContainSubstring, "ge-0/0/0")
}
