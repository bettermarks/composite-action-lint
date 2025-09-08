package compositeactionlint

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

const argv0 = "./composite-action-lint"

func TestCommandMain_Ok(t *testing.T) {

	files := []string{
		"./testdata/ok/single-action-step/action.yml",
		"./testdata/ok/single-shell-step/action.yml",
		"./testdata/ok/uses-inputs/action.yml",
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
