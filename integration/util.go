package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var baseDirectory string
var tempDirectory string
var coverageEnabled bool

func init() {
	_, filename, _, _ := runtime.Caller(1)
	baseDirectory = filepath.Join(filepath.Dir(filename), "..")
	if dir, err := ioutil.TempDir("", "fusty-test"); err != nil {
		panic(err)
	} else {
		tempDirectory = dir
	}
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.coverprofile=") {
			coverageEnabled = true
			break
		}
	}
}

func withTempConfig(conf *config.Config, f func(string)) {
	confFile, err := writeConfigFile(conf)
	if confFile != nil {
		defer os.Remove(confFile.Name())
	}
	So(err, ShouldBeNil)
	f(confFile.Name())
}

func writeConfigFile(conf *config.Config) (f *os.File, err error) {
	f, err = ioutil.TempFile("", "fusty-config")
	if err == nil {
		defer f.Close()
		if bytes, e := conf.ToBytes(); e != nil {
			err = e
		} else {
			_, err = f.Write(bytes)
		}
	}
	return
}

type fustyCmd struct {
	cmd fustyCmdAbstraction
}

type fustyCmdAbstraction interface {
	CombinedOutput() ([]byte, error)
	Success() bool
}

func runFusty(args ...string) *fustyCmd {
	if coverageEnabled {
		// TODO: &fustyCmd{cmd: &fustyCmdLocal{args: args}}
		// We really want this for code coverage
	}
	return &fustyCmd{cmd: &fustyCmdExternal{Cmd: exec.Command(filepath.Join(baseDirectory, "fusty"), args...)}}
}

type fustyCmdExternal struct {
	*exec.Cmd
}

func (f *fustyCmdExternal) CombinedOutput() ([]byte, error) {
	return f.Cmd.CombinedOutput()
}

func (f *fustyCmdExternal) Success() bool {
	So(f.Cmd.ProcessState, ShouldNotBeNil)
	return f.Cmd.ProcessState.Success()
}

type fustyCmdLocal struct {
}

func (f *fustyCmdLocal) CombinedOutput() ([]byte, error) {
	// Unfortunately, we can't properly capture stdout no matter what we try
	// probably due to goconvey intercepting state. Any help is welcome. Note,
	// to call main.Run, "gitlab.com/cretz/fusty" must be imported.
	panic("Not implemented")
}

func (f *fustyCmdLocal) Success() bool {
	panic("Not implemented")
}
