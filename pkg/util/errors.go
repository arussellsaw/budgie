package util

import (
	"errors"
	"net/http"

	"github.com/monzo/slog"
)

type ErrHandler func(http.ResponseWriter, *http.Request) error

func (h ErrHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h(w, r); err != nil {
		code := http.StatusInternalServerError
		c := &ErrCode{}
		if errors.As(err, &c) {
			code = c.Code
		}
		slog.Error(r.Context(), err.Error())
		http.Error(w, err.Error(), code)
	}
}

func WrapCode(err error, code int) error {
	return &ErrCode{Err: err, Code: code}
}

type ErrCode struct {
	Code int
	Err  error
}

func (e *ErrCode) Error() string {
	if e.Err == nil {
		return "nil"
	}
	return e.Err.Error()
}
func (e *ErrCode) Unwrap() error {
	return e.Err
}
