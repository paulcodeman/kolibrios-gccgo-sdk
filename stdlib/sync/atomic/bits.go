// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package atomic

func AndInt32(addr *int32, mask int32) (old int32)
func OrInt32(addr *int32, mask int32) (old int32)
func AndUint32(addr *uint32, mask uint32) (old uint32)
func OrUint32(addr *uint32, mask uint32) (old uint32)
func AndInt64(addr *int64, mask int64) (old int64)
func OrInt64(addr *int64, mask int64) (old int64)
func AndUint64(addr *uint64, mask uint64) (old uint64)
func OrUint64(addr *uint64, mask uint64) (old uint64)
func AndUintptr(addr *uintptr, mask uintptr) (old uintptr)
func OrUintptr(addr *uintptr, mask uintptr) (old uintptr)
