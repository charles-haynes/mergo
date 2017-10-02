// Copyright 2013 Dario Castañé. All rights reserved.
// Copyright 2009 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Based on src/pkg/reflect/deepequal.go from official
// golang's stdlib.

package mergo

import (
	"reflect"
)

// Traverses recursively both values, assigning src's fields values to dst.
// The map argument tracks comparisons that have already been seen, which allows
// short circuiting on recursive types.
func deepMerge(dst, src reflect.Value, overwrite bool) error {
	mergeable := func(k reflect.Kind) bool {
		switch k {
		case reflect.Map, reflect.Slice:
			return true
		case reflect.Ptr, reflect.Interface:
			return false // for now
		}
		return false
	}

	mergeStructs := func(dst, src reflect.Value) error {
		for i, n := 0, dst.NumField(); i < n; i++ {
			if err := deepMerge(dst.Field(i), src.Field(i), overwrite); err != nil {
				return err
			}
		}
		return nil
	}

	mergeMaps := func(dst, src reflect.Value) error {
		// src.Type() == dst.Type()
		for _, key := range src.MapKeys() {
			srcElement := src.MapIndex(key)
			if !srcElement.IsValid() || isEmptyValue(srcElement) {
				continue
			}
			dstElement := dst.MapIndex(key)
			if !dstElement.IsValid() || isEmptyValue(dstElement) || overwrite {
				dst.SetMapIndex(key, srcElement)
				continue
			}
			if !srcElement.CanInterface() {
				continue
			}
			srcElement = reflect.ValueOf(srcElement.Interface())
			dstElement = reflect.ValueOf(dstElement.Interface())
			switch srcElement.Kind() {
			case reflect.Slice:
				if dstElement.Kind() != reflect.Slice {
					continue
				}
				dst.SetMapIndex(key, reflect.AppendSlice(dstElement, srcElement))
			case reflect.Struct, reflect.Ptr, reflect.Map:
				if err := deepMerge(dstElement, srcElement, overwrite); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if !src.IsValid() || !dst.IsValid() || isEmptyValue(src) {
		return nil
	}

	if src.Type() != dst.Type() && (!mergeable(src.Kind()) || !mergeable(dst.Kind())) {
		return ErrDifferentArgumentsTypes
	}

	switch dst.Kind() {
	case reflect.Struct:
		return mergeStructs(dst, src)
	case reflect.Map:
		return mergeMaps(dst, src)
	case reflect.Ptr, reflect.Interface:
		if !overwrite && !isEmptyValue(dst) {
			return deepMerge(dst.Elem(), src.Elem(), overwrite)
		}
	case reflect.Slice:
		if dst.CanSet() && !overwrite && !isEmptyValue(dst) {
			dst.Set(reflect.AppendSlice(dst, src))
			return nil
		}
	}
	if dst.CanSet() && (overwrite || isEmptyValue(dst)) {
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
	return deepMerge(vDst, vSrc, overwrite)
}
