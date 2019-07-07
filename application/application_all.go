// Copyright (c) 2019, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//+build !windows

package application

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"syscall"
	"time"
)

func (a App) handleSignals() {
	signalChannel := make(chan os.Signal)
	signal.Notify(signalChannel, syscall.SIGUSR1)
	go func() {
	Loop:
		for {
			sig := <-signalChannel
			logger.Noticef("received signal: %v", sig)
			switch sig {
			case syscall.SIGUSR1:
				var dir string
				if a.homeDir != "" {
					dir = filepath.Join(a.homeDir, "debug", time.Now().UTC().Format("2006-01-02_15.04.05.000000"))
					if err := os.MkdirAll(dir, DefaultDirectoryMode); err != nil {
						logger.Errorf("debug dump: create debug log dir: %s", err)
						continue Loop
					}
				}

				if dir != "" {
					f, err := os.OpenFile(filepath.Join(dir, "vars"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, DefaultFileMode)
					if err != nil {
						logger.Errorf("debug dump: create vars file: %s", err)
						continue
					}
					if err := expvarExport(f); err != nil {
						logger.Errorf("debug dump: write vars file: %s", err)
					}
					if err := f.Close(); err != nil {
						logger.Errorf("debug dump: close vars file: %s", err)
					}
				} else {
					expvarExport(os.Stderr)
				}

				for _, d := range []struct {
					filename   string
					profile    string
					debugLevel int
				}{
					{
						filename:   "goroutine",
						profile:    "goroutine",
						debugLevel: 2,
					},
					{
						filename:   "heap",
						profile:    "heap",
						debugLevel: 0,
					},
					{
						filename:   "heap-verbose",
						profile:    "heap",
						debugLevel: 1,
					},
					{
						filename:   "block",
						profile:    "block",
						debugLevel: 1,
					},
					{
						filename:   "threadcreate",
						profile:    "threadcreate",
						debugLevel: 1,
					},
				} {
					if dir != "" {
						f, err := os.OpenFile(filepath.Join(dir, d.filename), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, DefaultFileMode)
						if err != nil {
							logger.Errorf("debug dump: create %s dump file: %s", d.filename, err)
							continue
						}
						pprof.Lookup(d.profile).WriteTo(f, d.debugLevel)
						if err := f.Close(); err != nil {
							logger.Errorf("debug dump: close %s file: %s", d.filename, err)
						}
					} else {
						fmt.Fprintln(os.Stderr, "debug dump:", d.filename)
						pprof.Lookup(d.profile).WriteTo(os.Stderr, d.debugLevel)
					}
				}
				if dir != "" {
					logger.Infof("debug dump: %s", dir)
				} else {
					logger.Info("debug dump: done")
				}
			}
		}
	}()
}
