// +build light

package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"log"
	"path/filepath"
	"testing"
	"time"
)

func TestControllerTLS(t *testing.T) {
	Convey("Given we are running the fusty controller command", t, func(c C) {
		Convey("When we start it with a custom public and private TLS key", func(c C) {
			/*
				The keys were created with (help from http://datacenteroverlords.com/2012/03/01/creating-your-own-ssl-certificate-authority/).
				https://security.stackexchange.com/questions/74345/provide-subjectaltname-to-openssl-directly-on-command-line explains
				whe we have to create a conf file instead of use the CLI.

				openssl genrsa -out tls_test_ca.key 2048
				openssl req -x509 -new -nodes -key tls_test_ca.key -days 10000 -out tls_test_ca.pem -subj "/C=US"
				openssl genrsa -out tls_test.key 2048
				openssl req -new -key tls_test.key -out tls_test.csr -subj "/C=US" -days 10000
				// Have to do this to be able to use an IP
				echo "subjectAltName = IP:127.0.0.1" > tls_test.cnf
				openssl x509 -req -in tls_test.csr -CA tls_test_ca.pem -CAkey tls_test_ca.key -CAcreateserial -out tls_test.pem -days 10000 -extfile tls_test.cnf
			*/

			ctx := newContext()
			ctx.initializeGitRepo(c)
			conf := ctx.newWorkingConfig()
			conf.Tls = &config.Tls{
				CertFile: filepath.Join(baseDirectory, "integration", "tls_test.pem"),
				KeyFile:  filepath.Join(baseDirectory, "integration", "tls_test.key"),
			}
			// We don't verify it's up since that does a ping
			controller := ctx.startControllerInBackground(c, conf, false)
			log.Print("Waiting 1 second for controller startup")
			time.Sleep(time.Duration(1) * time.Second)
			So(controller.Exited(), ShouldBeFalse)

			Convey("Then normal worker start should fail", func(c C) {
				worker := ctx.startWorkerInBackground(c)
				So(worker.Wait(3), ShouldBeNil)
				So(worker.Success(), ShouldBeFalse)
				So(string(controller.StreamedOutput()), ShouldContainSubstring, "TLS handshake error")
				So(string(worker.StreamedOutput()), ShouldContainSubstring, "malformed HTTP response")
			})

			Convey("Then a worker start asking for HTTPS without insecure setting should fail", func(c C) {
				worker := ctx.startWorkerInBackgroundWithArgs(c, "-controller", "https://127.0.0.1:9400")
				So(worker.Wait(3), ShouldBeNil)
				So(worker.Success(), ShouldBeFalse)
				So(string(controller.StreamedOutput()), ShouldContainSubstring, "TLS handshake error")
				So(string(worker.StreamedOutput()), ShouldContainSubstring, "certificate signed by unknown authority")
			})

			Convey("Then a worker start asking for HTTPS with insecure setting should succeed", func(c C) {
				worker := ctx.startWorkerInBackgroundWithArgs(c, "-controller", "https://127.0.0.1:9400", "-noverify")
				log.Print("Waiting 1 second for worker success")
				time.Sleep(time.Duration(1) * time.Second)
				So(worker.Exited(), ShouldBeFalse)
				So(string(controller.StreamedOutput()), ShouldNotContainSubstring, "TLS handshake error")
				So(string(worker.StreamedOutput()), ShouldNotContainSubstring, "certificate signed by unknown authority")
			})

			Convey("Then a worker start asking for HTTPS with custom CA should succeed", func(c C) {
				worker := ctx.startWorkerInBackgroundWithArgs(c, "-controller", "https://127.0.0.1:9400",
					"-cafile", filepath.Join(baseDirectory, "integration", "tls_test_ca.pem"))
				log.Print("Waiting 1 second for worker success")
				time.Sleep(time.Duration(1) * time.Second)
				So(worker.Exited(), ShouldBeFalse)
				So(string(controller.StreamedOutput()), ShouldNotContainSubstring, "TLS handshake error")
				So(string(worker.StreamedOutput()), ShouldNotContainSubstring, "certificate signed by unknown authority")
			})
		})
	})
}
