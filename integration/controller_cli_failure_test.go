package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"io/ioutil"
	"path/filepath"
	"testing"
)

func TestControllerFailures(t *testing.T) {
	Convey("Given we are running the fusty controller command", t, func() {

		Convey("When there is more than just a config flag", func() {
			cmd := runFusty("controller", "thisisnotconfig")
			cmd.conveyCommandFailure("only accepts single config-file")
		})

		Convey("When the config file cannot be found", func() {
			cmd := runFusty("controller", "-config", filepath.Join(baseDirectory, "doesnotexist.json"))
			cmd.conveyCommandFailure("Cannot find config file")
		})

		Convey("When we make a valid custom config", func() {
			conf := newWorkingConfig()

			Convey("When we are concerned with the job store configuration", func() {

				Convey("When we have no job store", func() {
					conf.JobStore = nil
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Job store configuration not found")
					})
				})

				Convey("When we change the job store type to invalid", func() {
					conf.JobStore.Type = "unknown"
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Unrecognized job store type: unknown")
					})
				})

				Convey("When we have an invalid generic", func() {
					conf.JobStore.JobStoreLocal.Jobs["bar"].Generic = "not-here"
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Unable to find job generic named: not-here")
					})
				})

				Convey("When we have an invalid generic schedule", func() {
					conf.JobStore.JobStoreLocal.JobGenerics["foo"].JobSchedule.Cron = "blah"
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Invalid schedule: missing field(s)")
					})
				})

				Convey("When we have an invalid schedule", func() {
					conf.JobStore.JobStoreLocal.Jobs["bar"].JobSchedule = &config.JobSchedule{Cron: "0 30 * * * *"}
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Invalid schedule: syntax error in hour field: '30'")
					})
				})

				Convey("When we have an empty command set", func() {
					conf.JobStore.JobStoreLocal.Jobs["bar"].JobCommand = &config.JobCommand{Inline: []string{}}
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Invalid command set: No commands in set")
					})
				})
			})

			Convey("When we are concerned with the device store configuration", func() {

				// TODO: test SSH defaults
				// TODO: test name becomes host if host not present

				Convey("When we have no device store", func() {
					conf.DeviceStore = nil
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Device store configuration not found")
					})
				})

				Convey("When we change the device store type to invalid", func() {
					conf.DeviceStore.Type = "unknown"
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Unrecognized device store type: unknown")
					})
				})

				Convey("When we have an invalid generic", func() {
					conf.DeviceStore.DeviceStoreLocal.Devices["qux"].Generic = "not-here"
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Unable to find device generic named: not-here")
					})
				})

				Convey("When we have an invalid protocol type", func() {
					conf.DeviceStore.DeviceStoreLocal.Devices["qux"].DeviceProtocol.Type = "unknown"
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Unrecognized protocol type: unknown")
					})
				})
			})

			Convey("When we are concerned with the data store configuration", func() {

				Convey("When we have no data store", func() {
					conf.DataStore = nil
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Data store configuration not found")
					})
				})

				Convey("When we change the data store type to invalid", func() {
					conf.DataStore.Type = "unknown"
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Unrecognized data store type: unknown")
					})
				})

				Convey("When there is no git URL", func() {
					conf.DataStore.DataStoreGit.Url = ""
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Data store for git requires url")
					})
				})

				Convey("When we use an unknown structure", func() {
					conf.DataStore.DataStoreGit.Structure = []string{"unknown"}
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Unrecognized git structure: unknown")
					})
				})

				Convey("When we use an invalid data directory", func() {
					conf.DataStore.DataStoreGit.DataDir = filepath.Join(tempDirectory, "notpresent")
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Failure obtaining git data directory")
					})
				})

				Convey("When we use an invalid email", func() {
					conf.DataStore.DataStoreGit.DataStoreGitUser.Email = "invalidemail"
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Invalid email for git user")
					})
				})

				Convey("When we use a password without user", func() {
					conf.DataStore.DataStoreGit.DataStoreGitUser.Pass = "somepass"
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("If git password supplied, username must also be supplied")
					})
				})

				Convey("When we use a git repository that doesn't exist", func() {
					dir, err := ioutil.TempDir(tempDirectory, "badgit")
					So(err, ShouldBeNil)
					conf.DataStore.DataStoreGit.Url = dir
					withTempConfig(conf, func(confFile string) {
						cmd := runFusty("controller", "-config", confFile)
						cmd.conveyCommandFailure("Git repository validation using ls-remote failed")
					})
				})
			})
		})
	})
}

func (c *fustyCmd) conveyCommandFailure(expectedString string) {
	Convey("Then the command should fail with '"+expectedString+"'", func() {
		out, _ := c.cmd.CombinedOutput()
		So(c.cmd.Success(), ShouldBeFalse)
		So(out, ShouldNotBeEmpty)
		Printf("FUSTY OUT: %v", string(out))
		So(string(out), ShouldContainSubstring, expectedString)
	})
}
