package compositeactionlint

import (
	"github.com/rhysd/actionlint"
)

type Error = actionlint.Error

// A positional agument constructor
func newError(message, filepath string, line, column int, kind string) *Error {
	return &Error{
		Message: message, Filepath: filepath, Line: line, Column: column, Kind: kind,
	}
}
