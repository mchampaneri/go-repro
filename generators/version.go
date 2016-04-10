package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	STATIC_VERSION = "0.1.1-dev"
	TEMPLATE       = `// THIS FILE WAS AUTOGENERATED BY GO GENEREATE. DO NOT EDIT!
package %s

func init() {
	version := %s
	dynamicVersion = &version
}
`
)

func quote(value string) string {
	return "`" + value + "`"
}

func runCommand(cmd *exec.Cmd) (stdout, stderr []byte, err error) {
	stdoutPipe, err := cmd.StdoutPipe()
	var stderrPipe io.ReadCloser
	stdoutBuf := bytes.Buffer{}
	stderrBuf := bytes.Buffer{}

	if err == nil {
		stderrPipe, err = cmd.StderrPipe()
	}

	if err == nil {
		err = cmd.Start()
	}

	if err == nil {
		wait := sync.WaitGroup{}
		errors := make(chan error)
		wait.Add(2)

		go func() {
			_, err := io.Copy(&stdoutBuf, stdoutPipe)
			if err != nil {
				errors <- err
			}

			wait.Done()
		}()

		go func() {
			_, err := io.Copy(&stderrBuf, stderrPipe)
			if err != nil {
				errors <- err
			}

			wait.Done()
		}()

		go func() {
			cmd.Start()
			wait.Wait()
			if err := cmd.Wait(); err != nil {
				errors <- err
			}
			close(errors)
		}()

		for e := range errors {
			if err == nil {
				err = e
			}
		}
	}

	return stdoutBuf.Bytes(), stderrBuf.Bytes(), err
}

func determineGitRev() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	stdout, stderr, err := runCommand(cmd)

	if err == nil {
		return strings.Trim(string(stdout), " \n")
	} else {
		fmt.Fprintln(os.Stderr, string(stdout))
		fmt.Fprintln(os.Stderr, string(stderr))
		fmt.Fprintln(os.Stderr, err)
		return "[unknown]"
	}
}

func version() string {
	gitRev := determineGitRev()
	now := time.Now()

	return fmt.Sprintf("%s (git:%s) (%s %s %s) (%s)",
		STATIC_VERSION,
		gitRev, runtime.Version(), runtime.GOOS, runtime.GOARCH,
		now.Format(time.RFC1123),
	)
}

func main() {
	var outfile, pkg string
	flag.StringVar(&outfile, "out", "", "output file")
	flag.StringVar(&pkg, "package", "", "target package")

	flag.Parse()

	if outfile == "" || pkg == "" {
		flag.Usage()
		os.Exit(1)
	}

	file, err := os.Create(outfile)
	if err != nil {
		panic(err)
	}

	_, err = fmt.Fprintf(file, TEMPLATE, pkg, quote(version()))
	if err != nil {
		panic(err)
	}

	err = file.Close()
	if err != nil {
		panic(err)
	}
}