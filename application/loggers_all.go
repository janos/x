// Copyright (c) 2019, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//+build !windows

package application

import (
	"log/syslog"

	"resenje.org/logging"
)

// NewSyslogHandler is a helper to easily create
// logging.SyslogHandler.
func NewSyslogHandler(facility logging.SyslogFacility, tag, network, address string) logging.Handler {
	if facility != "" {
		return &logging.SyslogHandler{
			Formatter: &logging.MessageFormatter{},
			Tag:       tag,
			Facility:  facility.Priority(),
			Severity:  syslog.Priority(logging.DEBUG),
			Network:   network,
			Address:   address,
		}
	}
	return nil
}
