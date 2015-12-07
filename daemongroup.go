// Copyright 2015 Pronovix
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package daemongroup

import (
	"fmt"
	"runtime/debug"
	"sync"
)

// Logger is a generic logger interface. This help the module not to be tied to the standard library's logger.
type Logger interface {
	Print(v ...interface{})
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

// Daemon is an interface that every type has to implement in order to be daemonized.
type Daemon interface {
	Start() error
}

type daemonData struct {
	Daemon
	Name    string
	Restart bool
}

// DaemonGroup manages daemons.
//
// If daemon panics or exists, it logs the result, and depending on the configuration,
// the daemon might be restarted.
type DaemonGroup struct {
	daemons []daemonData
	logger  Logger
}

// Creates a new daemon group with a logger.
func NewDaemonGroup(l Logger) *DaemonGroup {
	return &DaemonGroup{
		logger: l,
	}
}

// Adds a daemon to the DaemonGroup.
//
// This method is not thread-safe, do not call this after the daemon group started.
// TODO(tamasd): restart should be a number instead of a bool.
func (dg *DaemonGroup) AddDaemon(d Daemon, name string, restart bool) *DaemonGroup {
	dg.daemons = append(dg.daemons, daemonData{
		Daemon:  d,
		Name:    name,
		Restart: restart,
	})

	return dg
}

// Starts a DaemonGroup.
//
// This method blocks until all the daemons are finished running. If at least one of
// the daemons has restart enabled, this method will block forever.
func (dg *DaemonGroup) Start() error {
	var wg sync.WaitGroup

	wg.Add(len(dg.daemons))
	for _, d := range dg.daemons {
		go func(d daemonData) {
			defer wg.Done()
			for {
				if err := dg.startDaemon(d.Daemon, d.Name); err != nil {
					dg.logger.Println(err)
					if d.Restart {
						dg.logger.Printf("daemon %s failed, restarting...\n", d.Name)
						continue
					}
				}
				dg.logger.Printf("daemon %s stopped\n", d.Name)
				return
			}
		}(d)
	}

	wg.Wait()
	return nil
}

func (dg *DaemonGroup) startDaemon(d Daemon, name string) (err error) {
	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("daemon %s panic: %v\n", name, p)
			debug.PrintStack()
		}
	}()

	dg.logger.Printf("starting daemon %s...\n", name)
	err = d.Start()

	return
}
