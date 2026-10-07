// Validate ordinary typed provider errors without package-specific As hooks.
package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"sort"
	"syscall"
	"unsafe"

	"kos"
)

type providerError struct{ status int }

func (e *providerError) Error() string { return "provider response" }
func (e *providerError) Timeout() bool { return e.status == 504 }

type sliceError []int

func (sliceError) Error() string { return "uncomparable error" }

type customAsError struct{ value *providerError }

func (e customAsError) Error() string { return "custom conversion" }
func (e customAsError) As(target interface{}) bool {
	if p, ok := target.(**providerError); ok {
		*p = e.value
		return true
	}
	return false
}

func mustPanic(f func()) {
	panicked := false
	func() {
		defer func() { panicked = recover() != nil }()
		f()
	}()
	if !panicked {
		panic("invalid target accepted")
	}
}

func main() {
	kos.DebugString("OPENCODE_ERRORS_START")
	console, ok := kos.OpenConsole("OpenCode upstream errors and reflection test")
	if !ok {
		panic("console unavailable")
	}
	defer console.Close()
	original := &providerError{429}
	wrapped := fmt.Errorf("request: %w", original)
	var typed *providerError
	if !errors.As(wrapped, &typed) || typed != original || typed.status != 429 {
		panic("typed provider error extraction")
	}
	kos.DebugString("TYPED_PROVIDER_ERROR_OK")
	timeout := &providerError{504}
	var temporary interface{ Timeout() bool }
	if !errors.As(fmt.Errorf("request: %w", timeout), &temporary) || !temporary.Timeout() {
		panic("nonempty interface error extraction")
	}
	var plain error
	if !errors.As(wrapped, &plain) || plain != wrapped {
		panic("error interface extraction")
	}
	var any interface{}
	if !errors.As(wrapped, &any) || any != wrapped {
		panic("empty interface extraction")
	}
	joined := errors.Join(errors.New("first"), wrapped, timeout)
	if !errors.As(joined, &typed) || typed != original || !errors.Is(joined, timeout) {
		panic("joined error depth-first traversal")
	}
	if errors.Is(sliceError{1}, sliceError{1}) {
		panic("uncomparable error identity")
	}
	var sliced sliceError
	if !errors.As(errors.Join(nil, sliceError{7}), &sliced) || len(sliced) != 1 || sliced[0] != 7 {
		panic("uncomparable error value extraction")
	}
	if !errors.As(customAsError{timeout}, &typed) || typed != timeout {
		panic("custom As method")
	}
	if errors.As(nil, nil) {
		panic("nil error match")
	}
	mustPanic(func() { errors.As(original, nil) })
	mustPanic(func() { errors.As(original, typed) })
	var nilTarget *providerError
	mustPanic(func() { errors.As(original, nilTarget) })
	var number int
	mustPanic(func() { errors.As(original, &number) })
	kos.DebugString("UPSTREAM_ERRORS_TRAVERSAL_OK")
	rows := []struct {
		key     int
		name    string
		padding [3]int
	}{{3, "third", [3]int{30, 31, 32}}, {1, "first", [3]int{10, 11, 12}}, {2, "second", [3]int{20, 21, 22}}}
	sort.Slice(rows, func(i, j int) bool { return rows[i].key < rows[j].key })
	if rows[0].key != 1 || rows[0].name != "first" || rows[0].padding[2] != 12 || rows[2].key != 3 {
		panic("reflection swap of pointer-containing structures")
	}
	kos.DebugString("UPSTREAM_REFLECTLITE_SORT_OK")
	var integer int
	address := unsafe.Pointer(&integer)
	typeOfPointer := reflect.TypeOf(address)
	if typeOfPointer.Kind() != reflect.UnsafePointer || typeOfPointer.String() != "unsafe.Pointer" || !typeOfPointer.Comparable() || typeOfPointer.Size() != unsafe.Sizeof(address) {
		panic("upstream unsafe.Pointer descriptor")
	}
	if pointer := reflect.TypeOf(&address); pointer.Kind() != reflect.Pointer || pointer.String() != "*unsafe.Pointer" || pointer.Elem() != typeOfPointer {
		panic("upstream pointer-to-unsafe.Pointer descriptor")
	}
	var pointerValue interface{} = address
	if pointerValue != interface{}(address) {
		panic("unsafe.Pointer equality descriptor")
	}
	if !errors.Is(syscall.EACCES, fs.ErrPermission) || !errors.Is(syscall.EEXIST, fs.ErrExist) || !errors.Is(syscall.ENOENT, fs.ErrNotExist) || !errors.Is(syscall.ENOSYS, errors.ErrUnsupported) {
		panic("upstream filesystem errno identities")
	}
	if syscall.EAGAIN == syscall.EINVAL || !syscall.EAGAIN.Temporary() || !syscall.ETIMEDOUT.Timeout() {
		panic("distinct retry and invalid-argument errors")
	}
	if err := syscall.Pipe2(make([]int, 2), 1); !errors.Is(err, syscall.EINVAL) {
		panic("native invalid-flag errno translation")
	}
	if _, err := syscall.Read(-1, make([]byte, 1)); !errors.Is(err, syscall.EBADF) {
		panic("native invalid-descriptor errno translation")
	}
	kos.DebugString("UNSAFE_DESCRIPTORS_NATIVE_ERRNO_OK")
	pathError := &os.PathError{Op: "open", Path: "missing", Err: syscall.ENOENT}
	var filePathError *fs.PathError
	if !errors.As(pathError, &filePathError) || filePathError != pathError || !os.IsNotExist(pathError) {
		panic("original os/fs PathError alias and errno classification")
	}
	if !os.IsPermission(&os.PathError{Op: "open", Path: "denied", Err: syscall.EACCES}) || !os.IsExist(&os.LinkError{Op: "link", Old: "old", New: "new", Err: syscall.EEXIST}) {
		panic("original OS permission and existence classification")
	}
	if !os.IsTimeout(os.NewSyscallError("receive", syscall.ETIMEDOUT)) || os.NewSyscallError("receive", nil) != nil {
		panic("original syscall timeout classification")
	}
	contextual := fmt.Errorf("outer: %w", pathError)
	if os.IsNotExist(contextual) || !errors.Is(contextual, fs.ErrNotExist) {
		panic("original distinction between historical OS helpers and errors.Is")
	}
	kos.DebugString("ORIGINAL_OS_ERROR_API_OK")
	kos.DebugString("OPENCODE_ERRORS_PASS")
	console.WriteString("OPENCODE_ERRORS_PASS\n")
}
