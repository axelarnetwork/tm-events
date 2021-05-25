package errors

import (
	"fmt"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

func Wrapper(format string) func(error) error {
	return func(err error) error {
		return fmt.Errorf(format+": %w", err)
	}
}

func WrapperF(format string) func(error, ...interface{}) error {
	return func(err error, args ...interface{}) error {
		a := []interface{}{err}
		a = append(a, args...)
		return fmt.Errorf(format+": %w", a...)
	}
}

func GetMethodWrapper(methodDescription string) func(error, string) error {
	return func(err error, description string) error {
		return sdkerrors.Wrap(sdkerrors.Wrap(err, description), methodDescription)
	}
}

func SetWrappedError(methodDescription string, methodErr *error) func(error, string) {
	return func(err error, description string) {
		_err := sdkerrors.Wrap(sdkerrors.Wrap(err, description), methodDescription)
		*methodErr = _err
	}
}

type ErrorFWrapper func(err error, args ...interface{}) error
type ErrorWrapper func(err error) error
