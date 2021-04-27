package errors

import "fmt"

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

type ErrorFWrapper func(err error, args ...interface{}) error
type ErrorWrapper func(err error) error
