// Copyright (c) 2019, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build !windows
// +build !windows

package application

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime/pprof"
	"syscall"
	"time"

	"golang.org/x/exp/slog"
)

func (a App) handleSignals(logger *slog.Logger) {
	signalChannel := make(chan os.Signal, 1)
	signal.Notify(signalChannel, syscall.SIGUSR1)
	go func() {
	Loop:
		for {
			sig := <-signalChannel
			logger.Info("received signal", "signal", sig)
			switch sig {
			case syscall.SIGUSR1:
				var dir string
				if a.homeDir != "" {
					dir = filepath.Join(a.homeDir, "debug", time.Now().UTC().Format("2006-01-02_15.04.05.000000"))
					if err := os.MkdirAll(dir, DefaultDirectoryMode); err != nil {
						logger.Error("debug dump: create debug log dir", err)
						continue Loop
					}
				}

				if dir != "" {
					f, err := os.OpenFile(filepath.Join(dir, "vars"), os.O_CREATE|os.O_WRONLY|os.O_TRUNC, DefaultFileMode)
					if err != nil {
						logger.Error("debug dump: create vars file", err)
						continue
					}
					if err := expvarExport(f); err != nil {
						logger.Error("debug dump: write vars file", err)
					}
					if err := f.Close(); err != nil {
						logger.Error("debug dump: close vars file", err)
					}
				} else {
					if err := expvarExport(os.Stderr); err != nil {
						logger.Error("debug dump: expvar export", err)
					}
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
							logger.Error("debug dump: create dump file", err, "filename", d.filename)
							continue
						}
						if err := pprof.Lookup(d.profile).WriteTo(f, d.debugLevel); err != nil {
							logger.Error("debug dump: write to file", err, "filename", d.filename)
						}
						if err := f.Close(); err != nil {
							logger.Error("debug dump: close file", err, "filename", d.filename)
						}
					} else {
						fmt.Fprintln(os.Stderr, "debug dump:", d.filename)
						if err := pprof.Lookup(d.profile).WriteTo(os.Stderr, d.debugLevel); err != nil {
							logger.Error("debug dump: write to file", err, "filename", d.filename)
						}
					}
				}
				if dir != "" {
					logger.Info("debug dump: done", "dir", dir)
				} else {
					logger.Info("debug dump: done")
				}
			}
		}
	}()
}
