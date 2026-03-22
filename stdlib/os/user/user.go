package user

import (
	"errors"
	"os"
)

type User struct {
	Uid      string
	Gid      string
	Username string
	Name     string
	HomeDir  string
}

type Group struct {
	Gid  string
	Name string
}

func Current() (*User, error) {
	username := os.Getenv("USER")
	if username == "" {
		username = os.Getenv("USERNAME")
	}
	if username == "" {
		username = "user"
	}
	home := os.Getenv("HOME")
	if home == "" {
		if wd, err := os.Getwd(); err == nil {
			home = wd
		}
	}
	return &User{
		Uid:      "1",
		Gid:      "1",
		Username: username,
		Name:     username,
		HomeDir:  home,
	}, nil
}

func LookupGroupId(gid string) (*Group, error) {
	if gid == "" {
		return nil, errors.New("user: empty gid")
	}
	return &Group{Gid: gid, Name: gid}, nil
}
