// Copyright 2026 KolibriOS Go SDK contributors.
// Use of this source code is governed by the BSD-style license in LICENSE.

package syscall

// KolibriOS syscall 70 does not expose hard links or symbolic links.
func Link(oldname, newname string) error    { return ENOSYS }
func Symlink(oldname, newname string) error { return ENOSYS }
