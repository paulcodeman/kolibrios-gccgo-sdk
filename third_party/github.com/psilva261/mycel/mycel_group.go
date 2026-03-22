package mycel

import (
	"os/user"
)

// Group is the KolibriOS compatibility fallback used by the upstream browser
// 9p layer. There is no distinct host group concept exposed through the SDK.
func Group(u *user.User) (string, error) {
	if u == nil {
		return "", nil
	}
	return u.Username, nil
}
