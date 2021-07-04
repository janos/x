// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package application

import (
	"resenje.org/x/config"
)

// Config holds the common information for options: name and
// directories from where to load values.
//
// Deprecated: Use resenje.org/x/config.Config.
type Config = config.Config

// NewConfig creates a new instance of Config.
//
// Deprecated: Use resenje.org/x/config.New.
var NewConfig = config.New

// Options defines methods that are required for options.
//
// Deprecated: Use resenje.org/x/config.Options.
type Options = config.Options
