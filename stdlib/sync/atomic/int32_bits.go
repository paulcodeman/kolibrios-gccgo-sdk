// Copyright 2022 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package atomic

func (x *Int32) And(mask int32) (old int32) { return AndInt32(&x.v, mask) }

func (x *Int32) Or(mask int32) (old int32) { return OrInt32(&x.v, mask) }
