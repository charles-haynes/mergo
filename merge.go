// Copyright 2013 Dario Castañé. All rights reserved.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on src/pkg/reflect/deepequal.go from official
// golang's stdlib.

package mergo

import (
	"fmt"
	"reflect"
)

var indent = 0

// Traverses recursively both values, assigning src's fields values to dst.
// The map argument tracks comparisons that have already been seen, which allows
// short circuiting on recursive types.
func deepMerge(dst, src reflect.Value, visited map[visit]bool, depth int, overwrite bool) error {
	mergeStructs := func(dst, src reflect.Value) error {
		for i, n := 0, dst.NumField(); i < n; i++ {
			if err := deepMerge(dst.Field(i), src.Field(i), visited, depth+1, overwrite); err != nil {
				return err
			}
		}
		return nil
	}

	mergeMaps := func(dst, src reflect.Value) error {
		// src.Type() == dst.Type()
		for _, key := range src.MapKeys() {
			srcElement := src.MapIndex(key)
			dstElement := dst.MapIndex(key)
			if !dstElement.IsValid() || isEmptyValue(dstElement) || overwrite {
				dst.SetMapIndex(key, srcElement)
				continue
			}
			// if srcElement is an unexported field, give up. We can't get the value.
			if !srcElement.CanInterface() {
				continue
			}
			srcElement = reflect.ValueOf(srcElement.Interface())
			dstElement = reflect.ValueOf(dstElement.Interface())
			// make a settable value to merge into
			d := reflect.New(dstElement.Type()).Elem()
			d.Set(dstElement)
			err := deepMerge(d, srcElement, visited, depth+1, overwrite)
			if err != nil {
				continue
			}
			dst.SetMapIndex(key, d)
		}
		return nil
	}

	if src.Type() != dst.Type() {
		return fmt.Errorf("src and dst must be same type (%s) != (%s)", src.Type().String(), dst.Type().String())
	}

	// Taken from reflect.DeepEqual
	// We want to avoid putting more in the visited map than we need to.
	// For any possible reference cycle that might be encountered,
	// hard(t) needs to return true for at least one of the types in the cycle.
	hard := func(k reflect.Kind) bool {
		switch k {
		case reflect.Map, reflect.Slice, reflect.Ptr, reflect.Interface:
			return true
		}
		return false
	}

	if dst.CanAddr() && src.CanAddr() && hard(dst.Kind()) {
		addr1 := dst.UnsafeAddr()
		addr2 := src.UnsafeAddr()
		// Unfortunately can't canonicalize because
		// unlike equal, merge is not transitive

		// Short circuit if references are already seen.
		typ := dst.Type()
		v := visit{addr1, addr2, typ}
		if visited[v] {
			return nil
		}

		// Remember for later.
		visited[v] = true
	}

	if !src.IsValid() || !dst.IsValid() || isEmptyValue(src) {
		return nil
	}

	if isEmptyValue(dst) {
		if dst.CanSet() {
			dst.Set(src)
		}
		return nil
	}

	switch dst.Kind() {
	case reflect.Struct:
		return mergeStructs(dst, src)
	case reflect.Map:
		return mergeMaps(dst, src)
	case reflect.Ptr, reflect.Interface:
		if !overwrite && !isEmptyValue(dst) {
			return deepMerge(dst.Elem(), src.Elem(), visited, depth+1, overwrite)
		}
	case reflect.Slice:
		if dst.CanSet() && !overwrite && !isEmptyValue(dst) {
			dst.Set(reflect.AppendSlice(dst, src))
			return nil
		}
	}
	if dst.CanSet() && overwrite {
		dst.Set(src)
	}
	return nil
}

// Merge will fill any empty for value type attributes on the dst struct using corresponding
// src attributes if they themselves are not empty. dst and src must be valid same-type structs
// and dst must be a pointer to struct.
// It won't merge unexported (private) fields and will do recursively any exported field.
func Merge(dst, src interface{}) error {
	return merge(dst, src, false)
}

// MergeWithOverwrite will do the same as Merge except that non-empty dst attributes will be overriden by
// non-empty src attribute values.
func MergeWithOverwrite(dst, src interface{}) error {
	return merge(dst, src, true)
}

func merge(dst, src interface{}, overwrite bool) error {
	var (
		vDst, vSrc reflect.Value
		err        error
	)
	if vDst, vSrc, err = resolveValues(dst, src); err != nil {
		return err
	}
	if vDst.Type() != vSrc.Type() {
		return ErrDifferentArgumentsTypes
	}
	return deepMerge(vDst, vSrc, make(map[visit]bool), 0, overwrite)
}
