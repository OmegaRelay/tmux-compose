package main

import (
	"log"
	"sync"
	"time"
)

// A Session is ready when all it's windows are ready.
// A window is ready when all it's panes are ready.
// A pane is ready when the readycheck is successful.
// A pane without a readycheck is always ready.

type Object struct {
	Path       string `yaml:"-"`
	Name       string
	ReadyCheck struct {
		Test     string
		Interval time.Duration
		Retries  int
	}
	DependsOn []string `yaml:"depends_on"`
	mutex     sync.Mutex
	ready     bool
}

var allRunners []Runner
var byPath = make(map[string]Runner)

type Runner interface {
	GetObject() *Object
	DependenciesReady() bool
	IsReady() bool
	MarkReady()
	DoReadyCheck()
	Run()
}

func (o *Object) GetObject() *Object {
	return o
}

func (o *Object) DependenciesReady() bool {
	for _, path := range o.DependsOn {
		other := byPath[path]
		if !other.IsReady() {
			return false
		}
	}

	return true
}

func (o *Object) IsReady() bool {
	o.mutex.Lock()
	defer o.mutex.Unlock()
	return o.ready
}

func (o *Object) MarkReady() {
	o.mutex.Lock()
	o.ready = true
	o.mutex.Unlock()
}

func addRunner(r Runner) {
	allRunners = append(allRunners, r)

	path := r.GetObject().Path
	if path == "" {
		return
	}
	if runner, ok := byPath[path]; ok {
		log.Fatalf("Duplicate path: '%s' %v %v", path, runner, byPath)
	}
	byPath[path] = r
}

func (o *Object) Validate() {
	for _, path := range o.DependsOn {
		if _, ok := byPath[path]; !ok {
			log.Fatalf("Dependency does not exist: %s", path)
		}
	}
}

func validateDependencies() {
	for _, r := range allRunners {
		r.GetObject().Validate()
	}
}

func runAll() {
	validateDependencies()

	var wg sync.WaitGroup

	for _, r := range allRunners {
		// Don't use loop variables in goroutine
		wg.Add(1)

		go func(r Runner) {
			for !r.DependenciesReady() {
				time.Sleep(10 * time.Millisecond)
			}
			r.Run()
			r.DoReadyCheck()
			r.MarkReady()

			wg.Done()
		}(r)
	}

	wg.Wait()
}
