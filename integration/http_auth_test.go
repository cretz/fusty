// +build light

package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"log"
	"testing"
	"time"
)

func TestControllerHTTPAuth(t *testing.T) {
	Convey("Given we are running the fusty controller command", t, func(c C) {
		ctx := newContext()
		ctx.initializeGitRepo(c)
		conf := ctx.newWorkingConfig()

		Convey("When we start it with only user", func(c C) {
			conf.Username = "someuser"
			Convey("Then it fails to start", func(c C) {
				cont := ctx.startControllerInBackground(c, conf, false)
				So(cont.Wait(5), ShouldBeNil)
				So(cont.Success(), ShouldBeFalse)
				So(string(cont.StreamedOutput()), ShouldContainSubstring,
					"Username and password must be supplied together")
			})
		})

		Convey("When we start it with only password", func(c C) {
			conf.Password = "somepass"
			Convey("Then it fails to start", func(c C) {
				cont := ctx.startControllerInBackground(c, conf, false)
				So(cont.Wait(5), ShouldBeNil)
				So(cont.Success(), ShouldBeFalse)
				So(string(cont.StreamedOutput()), ShouldContainSubstring,
					"Username and password must be supplied together")
			})
		})

		Convey("When we start it with user/pass", func(c C) {
			conf.Username = "someuser"
			conf.Password = "somepass"
			cont := ctx.startControllerInBackground(c, conf, false)
			log.Print("Waiting 2 second for controller startup")
			time.Sleep(time.Duration(2) * time.Second)
			So(cont.Exited(), ShouldBeFalse)

			Convey("Then the worker cannot start normally", func(c C) {
				worker := ctx.startWorkerInBackground(c)
				So(worker.Wait(5), ShouldBeNil)
				So(worker.Success(), ShouldBeFalse)
				So(string(worker.StreamedOutput()), ShouldContainSubstring,
					"Bad status from /worker/ping: 401")
			})

			Convey("Then the worker can start with a bad user/pass", func(c C) {
				worker := ctx.startWorkerInBackgroundWithArgs(c, "-controller",
					"http://foo:bar@127.0.0.1:9400")
				So(worker.Wait(2), ShouldBeNil)
				So(string(worker.StreamedOutput()), ShouldContainSubstring, "Bad status from /worker/ping: 401")
			})

			Convey("Then the worker cannot start with user/pass", func(c C) {
				worker := ctx.startWorkerInBackgroundWithArgs(c, "-controller",
					"http://someuser:somepass@127.0.0.1:9400")
				So(worker.Wait(2), ShouldNotBeNil)
			})
		})

		Convey("When we start it with no user/pass", func(c C) {
			cont := ctx.startControllerInBackground(c, conf, false)
			log.Print("Waiting 2 second for controller startup")
			time.Sleep(time.Duration(2) * time.Second)
			So(cont.Exited(), ShouldBeFalse)

			Convey("Then the worker cannot start with user/pass", func(c C) {
				worker := ctx.startWorkerInBackgroundWithArgs(c, "-controller",
					"http://someuser:somepass@127.0.0.1:9400")
				So(worker.Wait(2), ShouldBeNil)
				So(worker.Success(), ShouldBeFalse)
				So(string(worker.StreamedOutput()), ShouldContainSubstring, "Bad status from /worker/ping: 403")
			})
		})
	})
}
