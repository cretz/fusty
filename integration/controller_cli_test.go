package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
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
			conf := config.NewDefault()

			Convey("When we are concerned with the job store configuration", func() {
				// Create a simple working job
				conf.JobStore = &config.JobStore{
					Type: "local",
					JobStoreLocal: &config.JobStoreLocal{
						JobGenerics: map[string]*config.Job{
							"foo": &config.Job{
								JobSchedule: &config.JobSchedule{
									Cron: "0 30 * * * *",
								},
								JobCommand: &config.JobCommand{
									Inline: []string{"command1", "command2"},
								},
							},
						},
						Jobs: map[string]*config.Job{
							"bar": &config.Job{
								Generic: "foo",
							},
						},
					},
				}

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
