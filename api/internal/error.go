package internal

import (
	"encoding/json"
	"net/http"

	"github.com/ledgerwatch/diagnostics"
)

type Error struct {
	Message string `json:"message"`
	Code    int    `json:"code"`
	Method  string `json:"method"`
	URI     string `json:"uri"`
}

func marshalError(r *http.Request, err error) Error {
	message := err.Error()

	var code int
	if diagnostics.IsNotFoundErr(err) {
		code = http.StatusUnauthorized
	} else if diagnostics.IsBadRequestErr(err) {
		code = http.StatusBadRequest

	} else {
		code = http.StatusInternalServerError
		message = err.Error()
	}

	return Error{
		Code:    code,
		Message: message,
		Method:  r.Method,
		URI:     r.URL.Path,
	}
}

func EncodeError(w http.ResponseWriter, r *http.Request, err error) {
	w.Header().Set("Content-Type", "application/json")
	encodedError := marshalError(r, err)
	w.WriteHeader(encodedError.Code)
	v := encodedError

	_ = json.NewEncoder(w).Encode(v)
}
