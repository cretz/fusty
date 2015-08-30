package controller

import (
	"errors"
	"gitlab.com/cretz/fusty/controller/config"
	"log"
	"path"
	"strconv"
	"sync"
	"time"
)

type DataStore interface {
	Store(job *DataStoreJob)
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

const jobKeySplit = "\x07"

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
	// TODO: validate each worker?
	dataStore := &gitDataStore{
		conf:                   conf,
		writesLock:             &sync.Mutex{},
		pendingWrites:          make(map[string][]*DataStoreJob),
		pendingWorkChan:        make(chan bool),
		runningWriteIds:        make(map[string]map[string]bool),
		waitingOnRunningWrites: make(map[string][]*DataStoreJob),
	}
	for i := 0; i < conf.PoolSize; i++ {
		worker := &gitWorker{
			initialized: false,
			dir:         path.Join(conf.DataDir, "pool"+strconv.Itoa(i+1)),
			dataStore:   dataStore,
		}
		go worker.run()
	}
	return nil, errors.New("Not implemented")
}

func (g *gitDataStore) Store(job *DataStoreJob) {
	// TODO: queue up readme overview...
	// Queue up the write
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
	jobs := make([]*DataStoreJob, len(g.pendingWrites), len(g.pendingWrites))
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
	initialized bool
	dir         string
	dataStore   *gitDataStore
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
	if err := g.initialize(); err != nil {
		log.Printf("Unable to initialize repository at %v: %v", g.dir, err)
		return
	}
	for _, job := range jobs {
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
	return errors.New("Not implemented")
}

func (g *gitWorker) commitJob(job *DataStoreJob) error {
	return errors.New("Not implemented")
}

func (g *gitWorker) push() error {
	return errors.New("Not implemented")
}
