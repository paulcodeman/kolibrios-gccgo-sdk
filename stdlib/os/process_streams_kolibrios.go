//go:build kolibrios && gccgo

package os

import (
	"io"
	"kos"
	"syscall"
)

// KolibriOS's loader has no inherited file table. Keep the original Go Cmd
// pipe and copying logic, and bridge its Files to native kernel socketpairs.
// Every child gets separate endpoints: Cmd closes its childIOFiles after Start.
type nativeProcessStream struct {
	input    bool
	file     *File
	owned    bool
	endpoint *File
	child    uint32
	done     chan error
}

func prepareNativeProcessStreams(files []*File) ([]string, []*nativeProcessStream, error) {
	specifications := make([]string, 3)
	var streams []*nativeProcessStream
	for index, file := range files {
		if file == nil {
			continue
		}
		if file.closed {
			abortNativeProcessStreams(streams)
			return nil, nil, syscall.EBADF
		}
		if file.name == DevNull {
			specifications[index] = DevNull
			continue
		}
		input := index == 0
		if input && !file.readable || !input && !file.writable {
			abortNativeProcessStreams(streams)
			return nil, nil, syscall.EBADF
		}
		// CombinedOutput shares its stdout and stderr File. Preserve ordering
		// with one kernel stream and one bridge rather than competing copies.
		if index == 2 && files[1] != nil && (files[1] == file || file.localStream != nil && file.localStream == files[1].localStream) {
			specifications[index] = specifications[1]
			continue
		}
		parent, child, code := kos.CreateLocalSocketPair()
		if code != 0 {
			abortNativeProcessStreams(streams)
			return nil, nil, syscall.ENOMEM
		}
		stream := &nativeProcessStream{input: input, file: file, child: child,
			endpoint: newNativeLocalFile("child stream", parent, !input, input), done: make(chan error, 1)}
		streams = append(streams, stream)
		{
			var err error
			stream.file, err = file.cloneNativeChildFile()
			if err != nil {
				abortNativeProcessStreams(streams)
				return nil, nil, err
			}
			stream.owned = true
		}
		specifications[index] = kos.LocalSocketStartupArgument(child)
	}
	return specifications, streams, nil
}

func abortNativeProcessStreams(streams []*nativeProcessStream) {
	for _, stream := range streams {
		_ = kos.CloseLocalSocket(stream.child)
		_ = stream.endpoint.Close()
		if stream.owned {
			_ = stream.file.Close()
		}
	}
}

func (stream *nativeProcessStream) copy() {
	var err error
	if stream.input {
		_, err = io.Copy(stream.endpoint, stream.file)
	} else {
		_, err = io.Copy(stream.file, stream.endpoint)
	}
	_ = stream.endpoint.Close()
	if stream.owned {
		_ = stream.file.Close()
	}
	stream.done <- err
}

func (p *Process) finishNativeStreams(reported bool) error {
	var first error
	for _, stream := range p.nativeStreams {
		// Normal SDK exit closes the child descriptors before reporting status.
		// A killed/crashed child cannot run that hook. Socket numbers are monotonic
		// kernel identifiers, not addresses or process-local fd-table indexes.
		if !reported {
			_ = kos.CloseLocalSocket(stream.child)
		}
		if stream.input {
			_ = stream.endpoint.Close()
			if stream.owned {
				_ = stream.file.Close()
			}
		}
	}
	for _, stream := range p.nativeStreams {
		if stream.input {
			continue
		} // Caller-owned console reads may be blocking.
		if err := <-stream.done; err != nil && first == nil {
			first = err
		}
	}
	return first
}
