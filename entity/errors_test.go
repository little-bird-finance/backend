package entity

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldError(t *testing.T) {
	tests := map[string]struct {
		field           string
		code            string
		description     string
		err             error
		errorStr        string
		wantFieldErrors []FieldError
	}{
		"basic": {
			field:       "myfield",
			code:        "empty",
			description: "can't be empty",
			err:         nil,
			errorStr:    "myfield:[empty] can't be empty",
			wantFieldErrors: []FieldError{
				{"myfield", "empty", "can't be empty", nil},
			},
		},
		"multiple fields errors": {
			field:       "field1",
			code:        "code1",
			description: "desc1",
			err:         NewFieldError(nil, "field2", "code2", "desc2"),
			errorStr:    "field1:[code1] desc1;field2:[code2] desc2",
			wantFieldErrors: []FieldError{
				{"field1", "code1", "desc1", NewFieldError(nil, "field2", "code2", "desc2")},
				{"field2", "code2", "desc2", nil},
			},
		},
		"multiple fields errors with another errors": {
			field:       "field1",
			code:        "code1",
			description: "desc1",
			err:         NewFieldError(errors.New("error"), "field2", "code2", "desc2"),
			errorStr:    "field1:[code1] desc1;field2:[code2] desc2;error",
			wantFieldErrors: []FieldError{
				{"field1", "code1", "desc1", NewFieldError(errors.New("error"), "field2", "code2", "desc2")},
				{"field2", "code2", "desc2", errors.New("error")},
			},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			// assert := assert.New(t)
			got := NewFieldError(tc.err, tc.field, tc.code, tc.description)
			assert.Equal(t, tc.field, got.Field(), "field")
			assert.Equal(t, tc.code, got.Code(), "code")
			assert.Equal(t, tc.description, got.Description(), "description")
			assert.Equal(t, tc.err, got.Unwrap(), "err")
			assert.Equal(t, tc.errorStr, got.Error(), "error")
			//assert(t, tc.wantFieldErrors, len(UnwrapFieldErrors(got)), "errors")
			// assert(t, tc.wantFieldErrors, UnwrapFieldErrors(got), "errors")

		})
	}

	// t.Run("need to return nil if has no FielErrors", func(t *testing.T) {
	// 	Assert(t, 0, len(UnwrapFieldErrors(fmt.Errorf("%w", errors.New("error1")))), "")
	// })

}
