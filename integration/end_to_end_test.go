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

func TestSimpleEndToEnd(t *testing.T) {
	Convey("Given we are running a mock SSH server, do it all (TODO: break up)", t, func(c C) {
		// log.Print("Starting a device")
		// device := startDefaultDevice()
		// Reset(device.stop)

		// Initialize the git path
		log.Print("Reinitializing git")
		cleanAndReinitializeGitRepo(c)

		// This config already has the data store set up properly
		conf := newWorkingConfig()

		// Set up one job every 3 seconds
		conf.JobStore.JobStoreLocal.JobGenerics = map[string]*config.Job{}
		// Make it fetch the Juniper config
		conf.JobStore.JobStoreLocal.Jobs = map[string]*config.Job{
			"simple": &config.Job{
				JobSchedule: &config.JobSchedule{
					Cron: "*/3 * * * * * *",
				},
				Type: "file",
				JobFile: map[string]*config.JobFile{
					"/config/juniper.conf.gz": &config.JobFile{Compression: "gzip"},
				},
			},
		}

		conf.DeviceStore.DeviceStoreLocal.DeviceGenerics = map[string]*config.Device{}
		// Give it the Juniper VM
		conf.DeviceStore.DeviceStoreLocal.Devices = map[string]*config.Device{
			"local": &config.Device{
				Host: "127.0.0.1",
				DeviceProtocol: &config.DeviceProtocol{
					Type:              "ssh",
					DeviceProtocolSsh: &config.DeviceProtocolSsh{Port: 2222},
				},
				DeviceCredentials: &config.DeviceCredentials{
					User: "root",
					Pass: "Juniper",
				},
				Jobs: map[string]*config.Job{
					"simple": &config.Job{},
				},
			},
		}

		// Fire up the controller
		log.Print("Starting controller")
		var controllerCmd *fustyCmd
		defer func() {
			if controllerCmd != nil && !controllerCmd.Exited() {
				controllerCmd.Stop()
			}
		}()
		confFile, err := writeConfigFile(conf)
		So(err, ShouldBeNil)
		bytes, err := conf.ToBytesPretty()
		So(err, ShouldBeNil)
		log.Printf("Running controller with config and waiting 3 seconds to start: %v", string(bytes))
		controllerCmd = runFusty(c, "controller", "-config", confFile.Name(), "-verbose")
		go controllerCmd.RunAndStreamToOutput("Controller out: ")
		// Wait just a sec and confirm it's still running
		time.Sleep(time.Duration(3) * time.Second)
		So(controllerCmd.Exited(), ShouldBeFalse)

		// Fire up the worker
		log.Print("Starting worker")
		var workerCmd *fustyCmd
		defer func() {
			if workerCmd != nil && !workerCmd.Exited() {
				workerCmd.Stop()
			}
		}()
		args := []string{
			"worker",
			"-controller",
			"http://127.0.0.1:9400",
			// We'll sleep for 20 minutes, because basically the worker will fetch work right from
			// the beginning and we only want to check the first run
			"-sleep",
			"1200",
			"-verbose",
			// We give a max of 1 because we only care about 1 execution
			"-maxjobs",
			"1",
		}
		workerCmd = runFusty(c, args...)
		go workerCmd.RunAndStreamToOutput("Worker out: ")

		// Wait for 5 seconds and shut em down...
		log.Print("Waiting 5 seconds")
		time.Sleep(time.Duration(5) * time.Second)
		log.Print("Shutting down worker")
		c.So(workerCmd.Stop(), ShouldBeNil)
		log.Print("Shutting down controller")
		c.So(controllerCmd.Stop(), ShouldBeNil)
		log.Print("Now checking status codes")
		time.Sleep(time.Duration(1) * time.Second)
		So(controllerCmd.Exited(), ShouldBeTrue)
		So(workerCmd.Exited(), ShouldBeTrue)

		// Now check that the git repository has a commit like we expect
		// First, we have to do a fresh clone
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
		responseStatuses: map[string]int{
			"command1": 0,
			"command2": 0,
		},
	}
	So(device.listen(), ShouldBeNil)
	go device.acceptUntilError()
	return device
}
