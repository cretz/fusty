// +build heavy

package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestSimpleEndToEnd(t *testing.T) {
	Convey("Given we are running a mock SSH server, do it all (TODO: break up)", t, func() {
		device := startDefaultDevice()
		Reset(device.stop)

		// Initialize the git path
		cleanAndReinitializeGitRepo()

		// This config already has the data store set up properly
		conf := newWorkingConfig()

		// Set up one job every 3 seconds
		conf.JobStore.JobStoreLocal.Jobs = map[string]*config.Job{
			"simple": &config.Job{
				JobSchedule: &config.JobSchedule{
					Cron: "*/3 * * * * * *",
				},
				JobCommand: &config.JobCommand{
					Inline: []string{"command1", "command2"},
				},
			},
		}

		// Give it the device we just started
		conf.DeviceStore.DeviceStoreLocal.Devices = map[string]*config.Device{
			"local": &config.Device{
				Host: device.addr().IP.String(),
				DeviceProtocol: &config.DeviceProtocol{
					Type: "ssh",
					DeviceProtocolSsh: &config.DeviceProtocolSsh{
						Port: device.addr().Port,
					},
				},
				DeviceCredentials: &config.DeviceCredentials{
					User: device.username,
					Pass: device.password,
				},
				Jobs: map[string]*config.Job{
					"simple": &config.Job{},
				},
			},
		}

		// Fire up the controller (TODO: stream output of this)
		var controllerCmd *fustyCmd
		defer controllerCmd.cmd.Stop()
		go func() {
			withTempConfig(conf, func(confFile string) {
				controllerCmd = runFusty("controller", "-config", confFile)
				out, _ := c.cmd.CombinedOutput()
				Printf("FUSTY CONTROLLER OUT: %v", string(out))
				So(c.cmd.Success(), ShouldBeTrue)
			})
		}()

		// Fire up the worker
		var workerCmd *fustyCmd
		defer workerCmd.cmd.Stop()
		go func() {
			withTempConfig(conf, func(confFile string) {
				args := []string{
					"worker",
					"-controller",
					device.listener.Addr().String(),
					// We'll sleep for 20 minutes, because basically the worker will fetch work right from
					// the beginning and we only want to check the first run
					"-sleep",
					"1200",
				}
				controllerCmd = runFusty()
				out, _ := c.cmd.CombinedOutput()
				Printf("FUSTY WORKER OUT: %v", string(out))
				So(c.cmd.Success(), ShouldBeTrue)
			})
		}()

		// Wait for 5 seconds and shut em down...
		time.Sleep(time.Duration(5) * time.Second)
		workerCmd.cmd.Stop()
		controllerCmd.cmd.Stop()

		// Now check that the git repository has a commit like we expect
		authorName := runInDir(gitRepoDirectory, "git", "log", "-1", "--pretty=%an")
		So(authorName, ShouldEqual, "John Doe")
		authorEmail := runInDir(gitRepoDirectory, "git", "log", "-1", "--pretty=%ae")
		So(authorEmail, ShouldEqual, "jdoe@example.com")
		commitComment := runInDir(gitRepoDirectory, "git", "log", "-1", "--pretty=%B")
		So(commitComment, ShouldContainSubstring, "Job: simple\n")
		So(commitComment, ShouldContainSubstring, "Device: local\n")
		// TODO: Some extra validation of the values here?
		So(commitComment, ShouldContainSubstring, "Expected Run Date:")
		So(commitComment, ShouldContainSubstring, "Start Date:")
		So(commitComment, ShouldContainSubstring, "End On:")
		So(commitComment, ShouldContainSubstring, "Elapsed Time:")
		filesText := runInDir(gitRepoDirectory, "git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
		filesUpdated := strings.Split(filesText, "\n")
		// TODO: Fix this when checking for other types of git structures
		So(len(filesUpdated), ShouldEqual, 1)
		So(filesUpdated, ShouldContain, "by_device/local/simple")

		// Now read the the file and make sure it looks right
		fileBytes, err := ioutil.ReadFile(filepath.Join(gitRepoDirectory, "by_device/local/simple"))
		So(err, ShouldBeNil)
		So(string(fileBytes), ShouldContainSubstring, strings.Repeat("This is a command response\n", 50))
	})
}

func startDefaultDevice() *mockDevice {
	device := &mockDevice{
		username: "johndoe",
		password: "secretpass",
		responses: map[string]string{
			"command2": strings.Repeat("This is a command response\n", 50),
		},
	}
	mock, err := device.listen()
	So(err, ShouldBeNil)
	go device.acceptUntilError()
	return mock
}
