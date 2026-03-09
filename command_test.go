package compositeactionlint

import (
	"bytes"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

const argv0 = "./composite-action-lint"

func TestCommandMain_Ok(t *testing.T) {

	files := []string{
		"./testdata/ok/single-action-step/action.yml",
		"./testdata/ok/single-shell-step/action.yml",
		"./testdata/ok/uses-inputs/action.yml",
		"./testdata/ok/shellcheck-expression/action.yml",
	}

	for _, filepath := range files {
		t.Run(filepath, func(t *testing.T) {
			// Replace with t.Output() from go 1.25
			var testOut bytes.Buffer
			c := Command{Stdout: &testOut, Stderr: &testOut}
			exitCode := c.Main([]string{argv0, filepath})

			t.Log(testOut.String())
			assert.Equal(t, 0, exitCode)
		})
	}
}

func TestCommandMain_BadActions(t *testing.T) {

	files := []string{
		"./testdata/examples/uses-and-run-step/action.yml",
		"./testdata/examples/steps-in-js-action/action.yml",
	}

	for _, filepath := range files {
		t.Run(filepath, func(t *testing.T) {
			// Replace with t.Output() from go 1.25
			var testOut bytes.Buffer
			c := Command{Stdout: &testOut, Stderr: &testOut}
			exitCode := c.Main([]string{argv0, filepath})

			t.Log(testOut.String())
			assert.Equal(t, 1, exitCode)
		})
	}
}

func TestCommandMain_ShellcheckError(t *testing.T) {
	if _, err := exec.LookPath("shellcheck"); err != nil {
		t.Skip("shellcheck not found on PATH")
	}

	var testOut bytes.Buffer
	c := Command{Stdout: &testOut, Stderr: &testOut}
	exitCode := c.Main([]string{argv0, "./testdata/examples/shellcheck-error/action.yml"})

	t.Log(testOut.String())
	assert.Equal(t, 1, exitCode)
}
