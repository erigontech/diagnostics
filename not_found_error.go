package diagnostics

import "errors"

type notFoundErr struct {
	error
}

func NotFound() error {
	return notFoundErr{error: errors.New("not found")}
}

func AsNotFound(err error) error {
	return notFoundErr{error: err}
}

func (err notFoundErr) IsNotFoundErr() bool {
	return true
}

func IsNotFoundErr(err error) bool {
	var target interface {
		IsNotFoundErr() bool
	}
	return errors.As(err, &target) && target.IsNotFoundErr()
}
