package client

import (
	"errors"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

func Dial(network, addr string) (*Conn, error) {
	c, err := net.Dial(network, addr)
	if err != nil {
		return nil, err
	}
	return NewConn(c)
}

func DialService(service string) (*Conn, error) {
	service = strings.TrimSpace(service)
	if service == "" {
		return nil, errors.New("empty service name")
	}
	registryPath := filepath.Join(Namespace(), service)
	data, err := os.ReadFile(registryPath)
	if err != nil {
		return nil, err
	}
	addr := strings.TrimSpace(string(data))
	if addr == "" {
		return nil, errors.New("empty service address")
	}
	return Dial("tcp", addr)
}

func Mount(network, addr string) (*Fsys, error) {
	c, err := Dial(network, addr)
	if err != nil {
		return nil, err
	}
	fsys, err := c.Attach(nil, getuser(), "")
	if err != nil {
		c.Close()
	}
	return fsys, err
}

func MountService(service string) (*Fsys, error) {
	c, err := DialService(service)
	if err != nil {
		return nil, err
	}
	fsys, err := c.Attach(nil, getuser(), "")
	if err != nil {
		c.Close()
	}
	return fsys, err
}

func MountServiceAname(service, aname string) (*Fsys, error) {
	c, err := DialService(service)
	if err != nil {
		return nil, err
	}
	fsys, err := c.Attach(nil, getuser(), aname)
	if err != nil {
		c.Close()
	}
	return fsys, err
}

// Namespace returns the path to the name space directory.
func Namespace() string {
	if ns := strings.TrimSpace(os.Getenv("NAMESPACE")); ns != "" {
		return ns
	}
	base := "/tmp0/1"
	if u, err := user.Current(); err == nil && u != nil && u.Username != "" {
		return filepath.Join(base, "ns."+u.Username+".kolibrios")
	}
	if u := strings.TrimSpace(os.Getenv("USER")); u != "" {
		return filepath.Join(base, "ns."+u+".kolibrios")
	}
	return filepath.Join(base, "ns.kolibrios")
}
