// Copyright (c) 2017, Janoš Guljaš <janos@resenje.org>
// All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package datadump

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"
	"time"
)

// Interface defines method to retrieve data Dump. If ifModifiedSince
// is not nil and data is not changed since provided time,
// both return values, Dump and error, will be nil.
type Interface interface {
	DataDump(fn func(f File) (err error)) (err error)
}

// InterfaceFunc implements Interface as a single function that can
// be assigned.
type InterfaceFunc func(fn func(f File) (err error)) (err error)

// DataDump calls the type function.
func (f InterfaceFunc) DataDump(fn func(f File) (err error)) (err error) {
	return f(fn)
}

// File defines a structure that holds dump metadata and body as reader interface.
// Body must be closed after the read is done.
type File struct {
	Name        string
	ContentType string
	Length      int64
	ModTime     *time.Time
	Body        io.ReadCloser
}

// Logger defines methods required for logging.
type Logger interface {
	Infof(format string, a ...interface{})
	Errorf(format string, a ...interface{})
}

// stdLogger is a simple implementation of Logger interface
// that uses log package for logging messages.
type stdLogger struct{}

func (l stdLogger) Infof(format string, a ...interface{}) {
	log.Printf("INFO "+format, a...)
}

func (l stdLogger) Errorf(format string, a ...interface{}) {
	log.Printf("ERROR "+format, a...)
}

// Handler returns http.Handler that will call DataDump on every o field that
// implements Interface. If filePrefix is not blank Content-Disposition HTTP
// header will be added to the response. The response body will be the tar
// archive containing binary files named by the o fields that implement
// Interface. The provided interface can be a struct or a map with string keys
// and interface{} values that will be checked if they implement the Interface.
// If compression argument is set to true, the response will be compressed with
// gzip default options.
func Handler(o interface{}, filePrefix string, logger Logger, compress bool) http.Handler {
	if logger == nil {
		logger = stdLogger{}
	}
	kind := reflect.Indirect(reflect.ValueOf(o)).Kind()
	if kind != reflect.Struct && kind != reflect.Map {
		panic(fmt.Sprintf("data dump: interface is not a struct or map: %T", o))
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		logger.Infof("data dump: started")

		extension := "tar"
		var rw io.Writer = w
		if compress {
			gzw := gzip.NewWriter(rw)
			defer gzw.Close()

			extension = "tar.gz"
			rw = gzw
		}
		tw := tar.NewWriter(rw)
		defer tw.Close()

		var length int64

		if compress {
			w.Header().Set("Content-Type", "application/gzip")
		} else {
			w.Header().Set("Content-Type", "application/octet-stream")
		}
		if filePrefix != "" {
			w.Header().Set("Content-Disposition", `attachment; filename="`+strings.Join([]string{start.UTC().Format("2006-01-02T15-04-05Z0700"), filePrefix}, "_")+`.`+extension)
		}
		w.Header().Set("Date", start.UTC().Format(http.TimeFormat))

		newDumpFn := func(name string) func(f File) (err error) {
			return func(f File) (err error) {
				if f.Name == "" {
					return errors.New("file name can not be blank")
				}
				if f.Body == nil {
					return errors.New("file body can not be nil")
				}
				logger.Infof("data dump: dumping %s file %s", name, f.Name)
				header := &tar.Header{
					Name: f.Name,
					Mode: 0666,
					Size: f.Length,
				}
				if f.ModTime != nil {
					header.ModTime = *f.ModTime
				}
				if err := tw.WriteHeader(header); err != nil {
					return fmt.Errorf("write file header %s in tar: %v", f.Name, err)
				}

				n, err := io.Copy(tw, f.Body)
				defer f.Body.Close()
				if err != nil {
					return fmt.Errorf("write file data %s in tar: %v", f.Name, err)
				}
				length += n
				logger.Infof("data dump: read %d bytes of %s file %s", n, name, f.Name)
				return nil
			}
		}

		v := reflect.Indirect(reflect.ValueOf(o))

		switch v.Kind() {
		case reflect.Struct:
			for i := 0; i < v.NumField(); i++ {
				if !v.Field(i).CanInterface() {
					continue
				}
				if u, ok := v.Field(i).Interface().(Interface); ok {
					name := v.Type().Field(i).Name
					if err := u.DataDump(newDumpFn(name)); err != nil {
						logger.Errorf("data dump: %s: %v", name, err)
					}
				}
			}
		case reflect.Map:
			for _, k := range v.MapKeys() {
				name, ok := k.Interface().(string)
				if !ok {
					continue
				}
				u, ok := v.MapIndex(k).Interface().(Interface)
				if !ok {
					continue
				}
				if err := u.DataDump(newDumpFn(name)); err != nil {
					logger.Errorf("data dump: %s: %v", name, err)
				}
			}
		}

		logger.Infof("data dump: wrote %d bytes in %s", length, time.Since(start))
	})
}
