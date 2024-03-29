// Copyright (c) 2019, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build windows
// +build windows

package application

import (
	"log/slog"
)

func (a App) handleSignals(logger *slog.Logger) {}
