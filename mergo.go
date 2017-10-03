// Copyright 2013 Dario Castañé. All rights reserved.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on src/pkg/reflect/deepequal.go from official
// golang's stdlib.

package mergo

import (
	"errors"
	"reflect"
)

// Errors reported by Mergo when it finds invalid arguments.
var (
	ErrNilArguments                = errors.New("src and dst must not be nil")
	ErrDifferentArgumentsTypes     = errors.New("src and dst must be of same type")
	ErrNotSupported                = errors.New("only structs and maps are supported")
	ErrExpectedMapAsDestination    = errors.New("dst was expected to be a map")
	ErrExpectedStructAsDestination = errors.New("dst was expected to be a struct")
)

// Taken from reflect.DeepEqual
// During deepMerge, must keep track of all merges  that are
// in progress. The merge algorithm assumes that all
// merges in progress are complete when it reencounters them.
// Visited merges are stored in a map indexed by visit.
type visit struct {
	a1  uintptr
	a2  uintptr
	typ reflect.Type
}

// From src/pkg/encoding/json.
func isEmptyValue(v reflect.Value) bool {
	switch v.Kind() {
	case reflect.Array, reflect.Map, reflect.Slice, reflect.String:
		return v.Len() == 0
	case reflect.Bool:
		return !v.Bool()
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() == 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return v.Uint() == 0
	case reflect.Float32, reflect.Float64:
		return v.Float() == 0
	case reflect.Interface, reflect.Ptr, reflect.Chan, reflect.Func:
		return v.IsNil()
	}
	return false
}

func resolveValues(dst, src interface{}) (vDst, vSrc reflect.Value, err error) {
	if dst == nil || src == nil {
		err = ErrNilArguments
		return
	}
	vDst = reflect.ValueOf(dst).Elem()
	if vDst.Kind() != reflect.Struct && vDst.Kind() != reflect.Map {
		err = ErrNotSupported
		return
	}
	vSrc = reflect.ValueOf(src)
	// We check if vSrc is a pointer to dereference it.
	if vSrc.Kind() == reflect.Ptr {
		vSrc = vSrc.Elem()
	}
	return
}
