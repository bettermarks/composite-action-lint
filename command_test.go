package compositeactionlint

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const argv0 = "./composite-action-lint"

func TestCommandMain_Ok(t *testing.T) {

	files := []string{
		"./testdata/ok/single-action-step/action.yml",
		"./testdata/ok/single-shell-step/action.yml",
	}

	for _, filepath := range files {
		t.Run(filepath, func(t *testing.T) {
			c := Command{Stdout: t.Output(), Stderr: t.Output()}
			exitCode := c.Main([]string{argv0, filepath})

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
			c := Command{Stdout: t.Output(), Stderr: t.Output()}
			exitCode := c.Main([]string{argv0, filepath})

			assert.Equal(t, 1, exitCode)
		})
	}
}
