package diagnostics

import (
	"errors"
	"fmt"
)

type badRequest struct {
	err error
}

func BadRequest(err error) error {
	return badRequest{err: err}
}

func (e badRequest) Error() string {
	return fmt.Sprintf("bad request: %v", e.err)
}

func (e badRequest) Unwrap() error {
	return e.err
}

func IsBadRequestErr(err error) bool {
	return errors.Is(err, badRequest{})
}
