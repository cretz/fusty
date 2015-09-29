package controller

import (
	"bytes"
	"errors"
	"fmt"
	"gitlab.com/cretz/fusty/config"
	"log"
	"net/mail"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type DataStore interface {
	Store(job *DataStoreJob)
}

func NewDataStoreFromConfig(conf *config.DataStore) (DataStore, error) {
	switch conf.Type {
	case "git":
		if conf.DataStoreGit == nil {
			return nil, errors.New("Data store for \"git\" required")
		}
		return newGitDataStore(conf.DataStoreGit)
	default:
		return nil, fmt.Errorf("Unrecognized data store type: %v", conf.Type)
	}
}

type DataStoreJob struct {
	DeviceName string
	JobName    string
	JobTime    time.Time
	StartTime  time.Time
	EndTime    time.Time
	Failure    string
	// TODO: worries about this eating too much mem?
	// Problem is we can't store reader because HTTP request is long gone
	Contents []byte
}

const (
	jobKeySplit          = "\x07"
	GitStructureByDevice = "by_device"
	GitStructureByJob    = "by_job"
)

func (d *DataStoreJob) key() string {
	return d.DeviceName + jobKeySplit + d.JobName
}

func (d *DataStoreJob) id() string {
	return d.key() + jobKeySplit + strconv.FormatInt(d.JobTime.Unix(), 10)
}

type gitDataStore struct {
	conf            *config.DataStoreGit
	writesLock      *sync.Mutex
	pendingWrites   map[string][]*DataStoreJob
	pendingWorkChan chan bool
	// By job key then by job ID
	runningWriteIds        map[string]map[string]bool
	waitingOnRunningWrites map[string][]*DataStoreJob
}

func newGitDataStore(conf *config.DataStoreGit) (*gitDataStore, error) {
	dataStore := &gitDataStore{
		conf:                   conf,
		writesLock:             &sync.Mutex{},
		pendingWrites:          make(map[string][]*DataStoreJob),
		pendingWorkChan:        make(chan bool),
		runningWriteIds:        make(map[string]map[string]bool),
		waitingOnRunningWrites: make(map[string][]*DataStoreJob),
	}
	if err := dataStore.ValidateAndApplyDefaults(); err != nil {
		return nil, err
	}
	if Verbose {
		log.Printf("Creating %v git data store copies for the pool", conf.PoolSize)
	}
	for i := 0; i < conf.PoolSize; i++ {
		worker := &gitWorker{
			dir:       path.Join(conf.DataDir, "pool"+strconv.Itoa(i+1)),
			dataStore: dataStore,
		}
		if err := worker.initialize(); err != nil {
			return nil, err
		}
		go worker.run()
	}
	return dataStore, nil
}

func (g *gitDataStore) ValidateAndApplyDefaults() error {
	if g.conf.Url == "" {
		return errors.New("Data store for git requires url")
	}
	if g.conf.PoolSize == 0 {
		g.conf.PoolSize = 20
	}
	if len(g.conf.Structure) == 0 {
		g.conf.Structure = []string{GitStructureByDevice}
	} else {
		for _, structure := range g.conf.Structure {
			if structure != GitStructureByDevice && structure != GitStructureByJob {
				return fmt.Errorf("Unrecognized git structure: %v", structure)
			}
		}
	}
	if g.conf.DataDir == "" {
		if dir, err := os.Getwd(); err != nil {
			return fmt.Errorf("Unable to get current working directory: %v", err)
		} else {
			g.conf.DataDir = dir
		}
	} else if fi, err := os.Stat(g.conf.DataDir); err != nil {
		return fmt.Errorf("Failure obtaining git data directory %v", g.conf.DataDir)
	} else if !fi.IsDir() {
		return fmt.Errorf("Git data directory %v not a directory", g.conf.DataDir)
	}
	if g.conf.DataStoreGitUser != nil {
		if g.conf.DataStoreGitUser.Email != "" {
			if _, err := mail.ParseAddress(g.conf.DataStoreGitUser.Email); err != nil {
				return fmt.Errorf("Invalid email for git user: %v", err)
			}
		}
		if g.conf.DataStoreGitUser.Pass != "" && g.conf.DataStoreGitUser.Name == "" {
			return errors.New("If git password supplied, username must also be supplied")
		}
	}
	// We do a simple ping check here to see if the repository even exists
	_, err := doGitCmd("", g.username(), g.password(), nil, "ls-remote", g.conf.Url)
	if err != nil {
		return fmt.Errorf("Git repository validation using ls-remote failed to validate URL %v: %v", g.conf.Url, err)
	}
	return nil
}

func (g *gitDataStore) username() string {
	if g.conf.DataStoreGitUser != nil {
		return g.conf.DataStoreGitUser.Name
	}
	return ""
}

func (g *gitDataStore) password() string {
	if g.conf.DataStoreGitUser != nil {
		return g.conf.DataStoreGitUser.Pass
	}
	return ""
}

func (g *gitDataStore) Store(job *DataStoreJob) {
	// TODO: queue up readme overview...
	// Queue up the write
	if Verbose {
		log.Printf("Preparing to store job %v on %v at expected time of %v with contents:\n%v",
			job.JobName, job.DeviceName, job.JobTime, string(job.Contents))
	}
	g.writesLock.Lock()
	key := job.key()
	// First, if it's running right now we put it in the waiting section
	if _, ok := g.runningWriteIds[key]; ok {
		if existing, ok := g.waitingOnRunningWrites[key]; ok {
			g.waitingOnRunningWrites[key] = append(existing, job)
		} else {
			g.waitingOnRunningWrites[key] = []*DataStoreJob{job}
		}
	} else {
		// Otherwise, just add to pending writes
		if existing, ok := g.pendingWrites[key]; ok {
			g.pendingWrites[key] = append(existing, job)
		} else {
			g.pendingWrites[key] = []*DataStoreJob{job}
		}
	}
	g.writesLock.Unlock()
	g.pendingWorkChan <- true
}

func (g *gitDataStore) nextJobs() []*DataStoreJob {
	g.writesLock.Lock()
	defer g.writesLock.Unlock()
	// Copy them all and clear it out if they are not empty
	// Also mark them as running
	if len(g.pendingWrites) == 0 {
		return nil
	}
	jobs := []*DataStoreJob{}
	for key, jobList := range g.pendingWrites {
		jobs = append(jobs, jobList...)
		// Mark as running
		writeIdMap, alreadyThere := g.runningWriteIds[key]
		if !alreadyThere {
			writeIdMap = make(map[string]bool)
			g.runningWriteIds[key] = writeIdMap
		}
		for _, job := range jobList {
			writeIdMap[job.id()] = true
		}
	}
	g.pendingWrites = make(map[string][]*DataStoreJob)
	return jobs
}

func (g *gitDataStore) markJobsCompleted(jobs []*DataStoreJob) {
	g.writesLock.Lock()
	// Remove them from the running set and any others that were waiting on
	// ones running need to be added to pending
	anythingEnqueued := false
	for _, job := range jobs {
		key := job.key()
		// Unmark as writing
		if writeIdMap, ok := g.runningWriteIds[key]; ok {
			id := job.id()
			if _, ok := writeIdMap[id]; ok {
				if len(writeIdMap) == 1 {
					delete(g.runningWriteIds, key)
				} else {
					delete(writeIdMap, id)
				}
			}
		}
		// Enqueue the waiting ones
		for _, pending := range g.waitingOnRunningWrites[key] {
			anythingEnqueued = true
			if existing, ok := g.pendingWrites[key]; ok {
				g.pendingWrites[key] = append(existing, pending)
			} else {
				g.pendingWrites[key] = []*DataStoreJob{pending}
			}
		}
	}
	defer g.writesLock.Unlock()
	if anythingEnqueued {
		g.pendingWorkChan <- true
	}
}

type gitWorker struct {
	dir       string
	dataStore *gitDataStore
}

func (g *gitWorker) run() {
	for {
		<-g.dataStore.pendingWorkChan
		jobs := g.dataStore.nextJobs()
		if len(jobs) > 0 {
			g.pushJobs(jobs)
		}
		g.dataStore.markJobsCompleted(jobs)
	}
}

func (g *gitWorker) pushJobs(jobs []*DataStoreJob) {
	// TODO: what to do on git errors...sleep, die, re-queue, etc?
	if err := g.clean(); err != nil {
		log.Printf("Unable to clean repository at %v: %v", g.dir, err)
		return
	}
	for _, job := range jobs {
		if Verbose {
			log.Printf("Committing and pushing job %v for device %v", job.JobName, job.DeviceName)
		}
		if err := g.commitJob(job); err != nil {
			log.Printf("Failed to commit job %v for device %v: %v", job.JobName, job.DeviceName, err)
		}
	}
	if err := g.push(); err != nil {
		for _, job := range jobs {
			log.Printf("Failed to push job %v for device %v: %v", job.JobName, job.DeviceName, err)
		}
	}
}

func (g *gitWorker) initialize() error {
	// Create the directory if needed
	if err := os.MkdirAll(g.dir, os.ModePerm); err != nil {
		return fmt.Errorf("Unable to initialize worker: %v", err)
	}
	// TODO: this needs to be cleaned if they change repo info, right? Or can we ask them to delete data dir
	if _, err := os.Stat(filepath.Join(g.dir, ".git")); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("Unable to read .git at %v: %v", g.dir, err)
	} else if os.IsNotExist(err) {
		// We need to git clone in to the directory
		if Verbose {
			log.Printf("Cloning into %v", g.dir)
		}
		if out, err := g.doGitCmd("clone", g.dataStore.conf.Url, g.dir); err != nil {
			return fmt.Errorf("Unable to clone from %v: %v. Output:\n%v", g.dataStore.conf.Url, err, out)
		}
	} else if Verbose {
		log.Printf("Not cloning fresh into %v because .git folder already exists", g.dir)
	}
	return g.clean()
}

func (g *gitWorker) clean() error {
	if Verbose {
		log.Printf("Performing hard reset and pull in %v", g.dir)
	}
	// Simple reset and pull
	if out, err := g.doGitCmd("reset", "--hard"); err != nil {
		return fmt.Errorf("Unable to git reset --hard in %v: %v. Output:\n%v", g.dir, err, out)
	}
	if out, err := g.doGitCmd("pull"); err != nil {
		return fmt.Errorf("Unable to git pull in %v: %v. Output:\n%v", g.dir, err, out)
	}
	return nil
}

func (g *gitWorker) commitJob(job *DataStoreJob) error {
	if len(job.Contents) > 0 {
		// TODO: should I write contents if there was a failure? I fear that if I do, it might be wildly
		//	different from a success which will make the diffs break. But if I don't, where does the
		//	failure go (i.e. is it too big for the commit message)?
		// Make the write to each place based on what structures exist
		for _, structure := range g.dataStore.conf.Structure {
			switch structure {
			case GitStructureByDevice:
				path := "by_device/" + job.DeviceName + "/" + job.JobName
				if err := g.writeGitFile(path, job.Contents); err != nil {
					return fmt.Errorf("Unable to write job to %v: %v", path, err)
				}
			case GitStructureByJob:
				path := "by_job/" + job.JobName + "/" + job.DeviceName
				if err := g.writeGitFile(path, job.Contents); err != nil {
					return fmt.Errorf("Unable to write job to %v: %v", path, err)
				}
			default:
				return fmt.Errorf("Unrecognized structure: %v", structure)
			}
		}
	}
	// If git status w/ porcelain returns anything, we need to add
	if out, err := g.doGitCmd("status", "--porcelain"); err != nil {
		return fmt.Errorf("Unable to check git status on %v: %v. Output:\n%v", g.dir, err, out)
	} else if strings.TrimSpace(out) != "" {
		// Add everything
		if out, err := g.doGitCmd("add", "."); err != nil {
			return fmt.Errorf("Unable to do git add on %v: %v. Output:\n%v", g.dir, err, out)
		}
	}
	// Commit w/ decent message regardless of whether files changed
	failure := ""
	if job.Failure != "" {
		failure = fmt.Sprintf("\n* Failure: %v", job.Failure)
	}
	message := fmt.Sprintf(
		"* Job: %v\n"+
			"* Device: %v\n"+
			"* Expected Run Date: %v\n"+
			"* Start Date: %v\n"+
			"* End On: %v\n"+
			"* Elapsed Time: %v"+failure,
		job.JobName, job.DeviceName, job.JobTime.Format(time.ANSIC),
		job.StartTime.Format(time.ANSIC), job.EndTime.Format(time.ANSIC), job.EndTime.Sub(job.StartTime),
	)
	// We --allow-empty so we can commit a message even without contents/change
	args := []string{"commit", "--allow-empty", "-m", message}
	// We have to make the author as friendly name or username
	env := map[string]string{}
	if g.dataStore.conf.DataStoreGitUser != nil &&
		(g.dataStore.conf.DataStoreGitUser.FriendlyName != "" || g.dataStore.conf.DataStoreGitUser.Email != "") {
		args = append(args, "--author",
			g.dataStore.conf.DataStoreGitUser.FriendlyName+" <"+g.dataStore.conf.DataStoreGitUser.Email+">")
		if g.dataStore.conf.DataStoreGitUser.FriendlyName != "" {
			env["GIT_COMMITTER_NAME"] = g.dataStore.conf.DataStoreGitUser.FriendlyName
		}
		if g.dataStore.conf.DataStoreGitUser.Email != "" {
			env["GIT_COMMITTER_EMAIL"] = g.dataStore.conf.DataStoreGitUser.Email
		}
	}
	if Verbose {
		log.Printf("Committing from %v with message:\n%v", g.dir, message)
	}
	if out, err := g.doGitCmdWithEnv(env, args...); err != nil {
		return fmt.Errorf("Unable to do git commit on %v: %v. Output:\n%v", g.dir, err, out)
	}
	return nil
}

func (g *gitWorker) push() error {
	if Verbose {
		log.Printf("Pushing from %v", g.dir)
	}
	if out, err := g.doGitCmd("push"); err != nil {
		return fmt.Errorf("Unable to do git push on %v: %v. Output:\n%v", g.dir, err, out)
	}
	return nil
}

func (g *gitWorker) doGitCmd(args ...string) (string, error) {
	return doGitCmd(g.dir, g.dataStore.username(), g.dataStore.password(), nil, args...)
}

func (g *gitWorker) doGitCmdWithEnv(env map[string]string, args ...string) (string, error) {
	return doGitCmd(g.dir, g.dataStore.username(), g.dataStore.password(), env, args...)
}

func (g *gitWorker) writeGitFile(path string, contents []byte) error {
	fullPath := filepath.Join(g.dir, path)
	if Verbose {
		log.Printf("Writing to file %v", fullPath)
	}
	dir, _ := filepath.Split(fullPath)
	if err := os.MkdirAll(dir, os.FileMode(600)); err != nil {
		return err
	}
	file, err := os.OpenFile(fullPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(600))
	if err != nil {
		return err
	}
	_, err = file.Write(contents)
	return err
}

func doGitCmd(dir string, username string, password string, env map[string]string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if len(env) > 0 {
		cmd.Env = os.Environ()
		for k, v := range env {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	if Verbose {
		log.Printf("Running git command with args %v", args)
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	pipe, err := cmd.StdinPipe()
	if err != nil {
		return "", fmt.Errorf("Cannot open stdin: %v", err)
	}
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("Unable to run git: %v", err)
	}
	// TODO: this doesn't feel right, should I sleep before checking out, etc
	if username != "" && strings.HasSuffix(out.String(), "Username:") {
		_, err = pipe.Write([]byte(username + "\n"))
	}
	if password != "" && err == nil && strings.HasSuffix(out.String(), "Password:") {
		_, err = pipe.Write([]byte(password + "\n"))
	}
	if err == nil {
		// TODO: this needs to time out after so long
		err = cmd.Wait()
	}
	if err != nil {
		if dir == "" {
			return "", fmt.Errorf("Error running git: %v. Output:\n%v", err, out.String())
		} else {
			return "", fmt.Errorf("Error running git in %v: %v. Output:\n%v", dir, err, out.String())
		}
	}
	return out.String(), nil
}
