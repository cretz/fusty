package integration

import (
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
)

var baseDirectory string
var tempDirectory string
var gitRepoDirectory string
var gitPullDataDirectory string
var coverageEnabled bool

func newWorkingConfig() *config.Config {
	return &config.Config{
		JobStore: &config.JobStore{
			Type: "local",
			JobStoreLocal: &config.JobStoreLocal{
				JobGenerics: map[string]*config.Job{
					"foo": &config.Job{
						JobSchedule: &config.JobSchedule{
							Cron: "0 0 * * * * *",
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
		},
		DeviceStore: &config.DeviceStore{
			Type: "local",
			DeviceStoreLocal: &config.DeviceStoreLocal{
				DeviceGenerics: map[string]*config.Device{
					"baz": &config.Device{},
				},
				Devices: map[string]*config.Device{
					"qux": &config.Device{
						Host:    "127.0.0.1",
						Generic: "baz",
						DeviceProtocol: &config.DeviceProtocol{
							Type: "ssh",
						},
					},
				},
			},
		},
		DataStore: &config.DataStore{
			Type: "git",
			DataStoreGit: &config.DataStoreGit{
				Url:      gitRepoDirectory,
				PoolSize: 1,
				DataDir:  gitPullDataDirectory,
				DataStoreGitUser: &config.DataStoreGitUser{
					FriendlyName: "John Doe",
					Email:        "jdoe@example.com",
				},
			},
		},
	}
}

func TestMain(m *testing.M) {
	res := m.Run()
	tearDown()
	os.Exit(res)
}

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
	// TODO: initialize git
	if dir, err := ioutil.TempDir(tempDirectory, "git-repo-temp"); err != nil {
		panic(err)
	} else {
		gitRepoDirectory = dir
	}
	if dir, err := ioutil.TempDir(tempDirectory, "git-pull-temp"); err != nil {
		panic(err)
	} else {
		gitPullDataDirectory = dir
	}
}

func cleanAndReinitializeGitRepo(c C) {
	c.So(os.RemoveAll(gitRepoDirectory), ShouldBeNil)
	c.So(os.MkdirAll(gitRepoDirectory, os.ModePerm), ShouldBeNil)
	cmd := exec.Command("git", "init", "--bare", gitRepoDirectory)
	_, err := cmd.CombinedOutput()
	c.So(err, ShouldBeNil)
	// We have to create a master branch which means we have to clone, commit empty, push a master, and then delete
	tempDir := filepath.Join(tempDirectory, "gittemp")
	c.So(os.RemoveAll(tempDir), ShouldBeNil)
	defer os.RemoveAll(tempDir)
	c.So(os.MkdirAll(tempDir, os.ModePerm), ShouldBeNil)
	cmd = exec.Command("git", "clone", gitRepoDirectory, tempDir)
	cmd.Dir = tempDir
	_, err = cmd.CombinedOutput()
	c.So(err, ShouldBeNil)
	cmd = exec.Command("git", "commit", "--allow-empty", "-m", "Creating master branch")
	cmd.Dir = tempDir
	_, err = cmd.CombinedOutput()
	c.So(err, ShouldBeNil)
	cmd = exec.Command("git", "push", "origin", "master")
	cmd.Dir = tempDir
	_, err = cmd.CombinedOutput()
	c.So(err, ShouldBeNil)
}

func tearDown() {
	os.RemoveAll(tempDirectory)
}

func withTempConfig(c C, conf *config.Config, f func(string)) {
	confFile, err := writeConfigFile(conf)
	if confFile != nil {
		defer os.Remove(confFile.Name())
	}
	c.So(err, ShouldBeNil)
	f(confFile.Name())
}

func writeConfigFile(conf *config.Config) (f *os.File, err error) {
	f, err = ioutil.TempFile(tempDirectory, "fusty-config")
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
	fustyCmdAbstraction
}

type fustyCmdAbstraction interface {
	RunAndStreamToOutput(prefix string) error
	CombinedOutput() ([]byte, error)
	Exited() bool
	Success() bool
	Stop() error
}

func runFusty(c C, args ...string) *fustyCmd {
	if coverageEnabled {
		// TODO: &fustyCmd{cmd: &fustyCmdLocal{args: args}}
		// We really want this for code coverage
	}
	return &fustyCmd{
		fustyCmdAbstraction: createExternalCmd(c, exec.Command(filepath.Join(baseDirectory, "fusty"), args...)),
	}
}

type fustyCmdExternal struct {
	c C
	*exec.Cmd
	lock *sync.Mutex
}

func createExternalCmd(c C, cmd *exec.Cmd) *fustyCmdExternal {
	return &fustyCmdExternal{
		c:    c,
		Cmd:  cmd,
		lock: &sync.Mutex{},
	}
}

func (f *fustyCmdExternal) RunAndStreamToOutput(prefix string) error {
	out := &stdoutWriter{prefix: prefix}
	f.Cmd.Stdout = out
	f.Cmd.Stderr = out
	return f.Cmd.Run()
}

func (f *fustyCmdExternal) CombinedOutput() ([]byte, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.Cmd.CombinedOutput()
}

func (f *fustyCmdExternal) Exited() bool {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.Cmd.ProcessState != nil && f.Cmd.ProcessState.Exited()
}

func (f *fustyCmdExternal) Success() bool {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.c.So(f.Cmd.ProcessState, ShouldNotBeNil)
	return f.Cmd.ProcessState.Success()
}

func (f *fustyCmdExternal) Stop() error {
	f.lock.Lock()
	defer f.lock.Unlock()
	f.c.So(f.Cmd.Process, ShouldNotBeNil)
	return f.Cmd.Process.Kill()
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

func (f *fustyCmdLocal) Stop() error {
	panic("Not implemented")
}

func runInDir(dir string, cmd string, args ...string) string {
	command := exec.Command(cmd, args...)
	command.Dir = dir
	out, err := command.CombinedOutput()
	So(err, ShouldBeNil)
	return strings.TrimSpace(string(out))
}

type stdoutWriter struct {
	prefix string
}

func (s *stdoutWriter) Write(p []byte) (n int, err error) {
	log.Print(s.prefix + strings.TrimSpace(string(p)))
	return len(p), nil
}
