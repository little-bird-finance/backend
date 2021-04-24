package entity

import (
	"errors"
	"fmt"
)

var (
	ErrUnknown   = errors.New("unknown error")
	ErrBusiness  = errors.New("")
	ErrTechnical = errors.New("")
	ErrNotFound  = fmt.Errorf("%wnot found", ErrBusiness)
)

func NewFieldError(err error, field, code, description string) FieldError {
	return FieldError{
		field:       field,
		code:        code,
		description: description,
		err:         err,
	}
}

func UnwrapFieldErrors(err error) []FieldError {
	res := make([]FieldError, 0)
	var fieldErr FieldError
	if errors.As(err, &fieldErr) {
		for ; errors.As(err, &fieldErr); err = errors.Unwrap(fieldErr) {
			res = append(res, fieldErr)
		}

	}

	if len(res) == 0 {
		return nil
	}
	return res

}

type FieldError struct {
	field       string
	code        string
	description string
	err         error
}

func (f FieldError) Error() string {
	const (
		format = "%s:[%s] %s"
	)
	if f.err == nil {
		return fmt.Sprintf(format, f.field, f.code, f.description)
	}
	return fmt.Sprintf(format+";%s", f.field, f.code, f.description, f.err.Error())
}

func (f FieldError) Unwrap() error {
	return f.err
}

func (f FieldError) Field() string {
	return f.field
}

func (f FieldError) Code() string {
	return f.code
}

func (f FieldError) Description() string {
	return f.description
}
