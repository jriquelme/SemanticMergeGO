// +build itest

package main_test

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var newLine = "\n"

func init() {
	if runtime.GOOS == "windows" {
		newLine = "\r\n"
	}
}

type TestCase struct {
	Source         string
	Encoding       string
	Output         string
	ExpectedOutput string
}

func TestSmgoCli(t *testing.T) {
	cli := filepath.Join(os.Getenv("GOPATH"), "bin", "smgo-cli")
	if runtime.GOOS == "windows" {
		cli = cli + ".exe"
	}
	_, err := os.Stat(cli)
	require.Nil(t, err)

	testCases := []TestCase{
		{
			Source:         "testdata/simple_func.go",
			Encoding:       "UTF-8",
			Output:         filepath.Join(os.TempDir(), "simple_func.yaml"),
			ExpectedOutput: "testdata/simple_func.yaml",
		},
	}

	cmd := exec.Command(cli, "shell", "flag-file")
	defer os.Remove("flag-file")

	wc, err := cmd.StdinPipe()
	require.Nil(t, err)
	rc, err := cmd.StdoutPipe()
	require.Nil(t, err)
	scanner := bufio.NewScanner(rc)

	err = cmd.Start()
	require.Nil(t, err)

	errChan := make(chan error, 1)
	go func(errChan chan<- error) {
		defer close(errChan)
		for _, tc := range testCases {
			io.WriteString(wc, tc.Source+newLine)
			io.WriteString(wc, tc.Encoding+newLine)
			io.WriteString(wc, tc.Output+newLine)
			scan := scanner.Scan()
			if !scan {
				errChan <- scanner.Err()
				return
			}
			t.Logf("parsing %s: %s", tc.Source, scanner.Text())
		}
		io.WriteString(wc, "end"+newLine)
	}(errChan)

	err = cmd.Wait()
	require.Nil(t, err)

	err = <-errChan
	require.Nil(t, err)

	for _, tc := range testCases {
		output, err := ioutil.ReadFile(tc.Output)
		require.Nil(t, err)
		expectedOutput, err := ioutil.ReadFile(tc.ExpectedOutput)
		require.Nil(t, err)
		if !bytes.Equal(output, expectedOutput) {
			t.Errorf("mismatch between %s and %s", tc.Output, tc.ExpectedOutput)
		} else {
			err = os.Remove(tc.Output)
			assert.Nil(t, err)
		}
	}
}
