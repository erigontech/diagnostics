package diagnostics

import "errors"

type badRequest struct {
	error
}

func BadRequest() error {
	return badRequest{error: errors.New("bad request")}
}

func (err badRequest) IsBadRequestErr() bool {
	return true
}

func AsBadRequestErr(err error) error {
	return badRequest{error: err}
}

func IsBadRequestErr(err error) bool {
	var target interface {
		IsBadRequestErr() bool
	}
	return errors.As(err, &target) && target.IsBadRequestErr()
}
