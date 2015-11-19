package integration

import (
	"bytes"
	. "github.com/smartystreets/goconvey/convey"
	"gitlab.com/cretz/fusty/config"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"
)

var baseDirectory string
var globalTempDirectory string
var coverageEnabled bool

type context struct {
	tempDirectory        string
	gitRepoDirectory     string
	gitPullDataDirectory string
	verbose              bool
}

func newContext() *context {
	tempDirectory, err := ioutil.TempDir(globalTempDirectory, "fusty-test")
	So(err, ShouldBeNil)
	gitRepoDirectory, err := ioutil.TempDir(tempDirectory, "git-repo-temp")
	So(err, ShouldBeNil)
	gitPullDataDirectory, err := ioutil.TempDir(tempDirectory, "git-pull-temp")
	So(err, ShouldBeNil)
	return &context{
		tempDirectory:        tempDirectory,
		gitRepoDirectory:     gitRepoDirectory,
		gitPullDataDirectory: gitPullDataDirectory,
	}
}

func (ctx *context) newWorkingConfig() *config.Config {
	return &config.Config{
		Ip:   "127.0.0.1",
		Port: 9400,
		JobStore: &config.JobStore{
			Type: "local",
			JobStoreLocal: &config.JobStoreLocal{
				JobGenerics: map[string]*config.Job{
					"foo": &config.Job{
						JobSchedule: &config.JobSchedule{
							Cron: "0 0 * * * * *",
						},
						Commands: []*config.JobCommand{
							&config.JobCommand{
								Command: "command1",
							},
							&config.JobCommand{
								Command: "command2",
							},
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
				Url:      ctx.gitRepoDirectory,
				PoolSize: 1,
				DataDir:  ctx.gitPullDataDirectory,
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
		globalTempDirectory = dir
	}
	for _, arg := range os.Args {
		if strings.HasPrefix(arg, "-test.coverprofile=") {
			coverageEnabled = true
			break
		}
	}
}

func (ctx *context) initializeGitRepo(c C) {
	c.So(os.MkdirAll(ctx.gitRepoDirectory, os.ModePerm), ShouldBeNil)
	cmd := exec.Command("git", "init", "--bare", ctx.gitRepoDirectory)
	_, err := cmd.CombinedOutput()
	c.So(err, ShouldBeNil)
	// We have to create a master branch which means we have to clone, commit empty, push a master, and then delete
	tempDir := filepath.Join(ctx.tempDirectory, "gittemp")
	c.So(os.RemoveAll(tempDir), ShouldBeNil)
	defer os.RemoveAll(tempDir)
	c.So(os.MkdirAll(tempDir, os.ModePerm), ShouldBeNil)
	cmd = exec.Command("git", "clone", ctx.gitRepoDirectory, tempDir)
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
	os.RemoveAll(globalTempDirectory)
}

func (ctx *context) withTempConfig(c C, conf *config.Config, f func(string)) {
	confFile, err := ctx.writeConfigFile(conf)
	if err != nil {
		defer os.Remove(confFile)
	}
	c.So(err, ShouldBeNil)
	f(confFile)
}

func (ctx *context) writeConfigFile(conf *config.Config) (string, error) {
	f, err := ioutil.TempFile(ctx.tempDirectory, "fusty-config")
	filename := ""
	if err == nil {
		func() {
			defer f.Close()
			if bytes, e := conf.ToJSON(false); e != nil {
				err = e
			} else {
				_, err = f.Write(bytes)
			}
		}()
		if err == nil {
			// Move the file so it can have .json extension
			filename = f.Name() + ".json"
			err = os.Rename(f.Name(), filename)
		}
	}
	return filename, err
}

type fustyCmd struct {
	fustyCmdAbstraction
}

type fustyCmdAbstraction interface {
	RunAndStreamOutput(prefix string) error
	CombinedOutput() ([]byte, error)
	StreamedOutput() []byte
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

func (c *fustyCmd) conveyCommandFailure(expectedString string) {
	Convey("Then the command should fail with '"+expectedString+"'", func() {
		out, _ := c.CombinedOutput()
		So(c.Success(), ShouldBeFalse)
		So(out, ShouldNotBeEmpty)
		log.Printf("Fusty out: %v", string(out))
		So(string(out), ShouldContainSubstring, expectedString)
	})
}

type fustyCmdExternal struct {
	c C
	*exec.Cmd
	lock           *sync.Mutex
	streamedOutput *bytes.Buffer
}

func createExternalCmd(c C, cmd *exec.Cmd) *fustyCmdExternal {
	return &fustyCmdExternal{
		c:    c,
		Cmd:  cmd,
		lock: &sync.Mutex{},
	}
}

func (f *fustyCmdExternal) RunAndStreamOutput(prefix string) error {
	f.streamedOutput = &bytes.Buffer{}
	out := io.MultiWriter(&stdoutWriter{prefix: prefix}, f.streamedOutput)
	f.Cmd.Stdout = out
	f.Cmd.Stderr = out
	return f.Cmd.Run()
}

func (f *fustyCmdExternal) CombinedOutput() ([]byte, error) {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.Cmd.CombinedOutput()
}

func (f *fustyCmdExternal) StreamedOutput() []byte {
	f.lock.Lock()
	defer f.lock.Unlock()
	return f.streamedOutput.Bytes()
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
	log.Printf("Result of command %v with args %v:\n%v", cmd, args, string(out))
	return strings.TrimSpace(string(out))
}

type stdoutWriter struct {
	prefix string
}

func (s *stdoutWriter) Write(p []byte) (n int, err error) {
	log.Print(s.prefix + strings.TrimSpace(string(p)))
	return len(p), nil
}

func (ctx *context) startControllerInBackground(c C, conf *config.Config, verifyUp bool) *fustyCmd {
	log.Print("Starting controller")
	var controllerCmd *fustyCmd
	Reset(func() {
		if controllerCmd != nil && !controllerCmd.Exited() {
			controllerCmd.Stop()
		}
	})
	confFile, err := ctx.writeConfigFile(conf)
	c.So(err, ShouldBeNil)
	bytes, err := conf.ToJSON(true)
	c.So(err, ShouldBeNil)
	log.Printf("Running controller with config and waiting 3 seconds to start: %v", string(bytes))
	args := []string{"controller", "-config", confFile}
	if ctx.verbose {
		args = append(args, "-verbose")
	}
	controllerCmd = runFusty(c, args...)
	go controllerCmd.RunAndStreamOutput("Controller out: ")
	if !verifyUp {
		return controllerCmd
	}
	// Try once a second for 10 seconds to see if up
	url := "http://" + conf.Ip + ":" + strconv.Itoa(conf.Port) + "/worker/ping"
	success := false
	for i := 0; i < 5; i++ {
		time.Sleep(time.Second)
		if resp, err := http.Get(url); err == nil && resp.StatusCode == http.StatusOK {
			success = true
			break
		}
	}
	if !success {
		log.Printf("Unable to connect to controller at %v", url)
		controllerCmd.Stop()
		c.So(success, ShouldBeTrue)
	}
	c.So(controllerCmd.Exited(), ShouldBeFalse)
	return controllerCmd
}

func (ctx *context) startWorkerInBackground(c C) *fustyCmd {
	args := []string{
		"-controller",
		"http://127.0.0.1:9400",
		// We'll sleep for 20 minutes, because basically the worker will fetch work right from
		// the beginning and we only want to check the first run
		"-sleep",
		"1200",
		// We give a max of 1 because we only care about 1 execution
		"-maxjobs",
		"1",
	}
	return ctx.startWorkerInBackgroundWithArgs(c, args...)
}

func (ctx *context) startWorkerInBackgroundWithArgs(c C, args ...string) *fustyCmd {
	log.Print("Starting worker")
	var workerCmd *fustyCmd
	Reset(func() {
		if workerCmd != nil && !workerCmd.Exited() {
			workerCmd.Stop()
		}
	})
	args = append([]string{"worker"}, args...)
	if ctx.verbose {
		args = append(args, "-verbose")
	}
	workerCmd = runFusty(c, args...)
	go workerCmd.RunAndStreamOutput("Worker out: ")
	return workerCmd
}

type gitAssertion struct {
	job          string
	device       string
	filesUpdated []string
	fileContents string
}

func (g *gitAssertion) assertValid(ctx *context) {
	gitAssertDir, err := ioutil.TempDir(ctx.tempDirectory, "git-assert-temp")
	So(err, ShouldBeNil)
	So(os.MkdirAll(gitAssertDir, os.ModePerm), ShouldBeNil)
	runInDir(gitAssertDir, "git", "clone", ctx.gitRepoDirectory, gitAssertDir)

	authorName := runInDir(gitAssertDir, "git", "log", "-1", "--pretty=%an")
	So(authorName, ShouldEqual, "John Doe")
	authorEmail := runInDir(gitAssertDir, "git", "log", "-1", "--pretty=%ae")
	So(authorEmail, ShouldEqual, "jdoe@example.com")
	commitComment := runInDir(gitAssertDir, "git", "log", "-1", "--pretty=%B")
	So(commitComment, ShouldContainSubstring, "Job: "+g.job+"\n")
	So(commitComment, ShouldContainSubstring, "Device: "+g.device+"\n")
	// TODO: Some extra validation of the values here?
	So(commitComment, ShouldContainSubstring, "Expected Run Date:")
	So(commitComment, ShouldContainSubstring, "Start Date:")
	So(commitComment, ShouldContainSubstring, "End On:")
	So(commitComment, ShouldContainSubstring, "Elapsed Time:")
	filesText := runInDir(gitAssertDir, "git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
	filesUpdated := strings.Split(filesText, "\n")
	// TODO: Fix this when checking for other types of git structures
	So(len(filesUpdated), ShouldEqual, len(g.filesUpdated))
	for _, fileUpdated := range g.filesUpdated {
		So(filesUpdated, ShouldContain, fileUpdated)
	}

	// Now read the the file and make sure it looks right
	fileBytes, err := ioutil.ReadFile(filepath.Join(gitAssertDir, g.filesUpdated[0]))
	So(err, ShouldBeNil)
	expected := strings.Replace(strings.TrimSpace(g.fileContents), "\r\n", "\n", -1)
	actual := strings.Replace(strings.TrimSpace(string(fileBytes)), "\r\n", "\n", -1)
	// Change /r/n to /n
	So(actual, ShouldContainSubstring, expected)
}
