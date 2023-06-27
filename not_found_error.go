package diagnostics

import (
	"errors"
	"fmt"
)

type notFoundErr struct {
	err error
}

func NotFound(err error) error {
	return notFoundErr{err: err}
}

func (e notFoundErr) Error() string {
	return fmt.Sprintf("not found: %v", e.err)
}

func (e notFoundErr) Unwrap() error {
	return e.err
}

func IsNotFoundErr(err error) bool {
	return errors.Is(err, notFoundErr{})
}
