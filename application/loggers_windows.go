// Copyright (c) 2019, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//+build windows

package application

import "resenje.org/logging"

// NewSyslogHandler is not supported on windows.
func NewSyslogHandler(facility logging.SyslogFacility, tag, network, address string) logging.Handler {
	return nil
}
