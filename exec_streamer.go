package execstreamer

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os/exec"
)

//ExecStreamer is the streamer interface (built by the ExecStreamerBuilder)
type ExecStreamer interface {
	StartExec() (*exec.Cmd, error)
	ExecAndWait() error
}

type execStreamer struct {
	ExecutorName string
	Exe          string
	Args         []string
	Dir          string
	Env          []string

	StdoutWriter io.Writer
	StdoutPrefix string

	StderrWriter io.Writer
	StderrPrefix string

	AutoFlush bool
}

func (e *execStreamer) flushIfEnabled(writer io.Writer) {
	if e.AutoFlush {
		if flusher, ok := writer.(http.Flusher); ok {
			flusher.Flush()
		}
	}
}

//StartExec will execute the command using the given executor and return (without waiting for completion) with the exec.Cmd
func (e *execStreamer) StartExec() (*exec.Cmd, error) {
	x, err := NewExecutorFromName(e.ExecutorName)
	if err != nil {
		return nil, err
	}

	cmd := x.GetCommand(e.Exe, e.Args...)

	if e.Dir != "" {
		cmd.Dir = e.Dir
	}
	if len(e.Env) > 0 {
		cmd.Env = e.Env
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}

	stdoutScanner := bufio.NewScanner(stdout)
	go func() {
		for stdoutScanner.Scan() {
			fmt.Fprintf(e.StdoutWriter, "%s%s\n", e.StdoutPrefix, stdoutScanner.Text())
			e.flushIfEnabled(e.StdoutWriter)
		}
	}()

	stderrScanner := bufio.NewScanner(stderr)
	go func() {
		for stderrScanner.Scan() {
			fmt.Fprintf(e.StderrWriter, "%s%s\n", e.StderrPrefix, stderrScanner.Text())
			e.flushIfEnabled(e.StderrWriter)
		}
	}()

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	return cmd, nil
}

//ExecAndWait will execute the command using the given executor and wait until completion
func (e *execStreamer) ExecAndWait() error {
	cmd, err := e.StartExec()
	if err != nil {
		return err
	}

	err = cmd.Wait()
	if err != nil {
		return err
	}

	return nil
}
