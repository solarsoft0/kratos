package sms

import (
	"github.com/ory/jsonschema/v3"
	"github.com/ory/kratos/schema"
	"github.com/ory/kratos/text"
	"net/http"
)

type ValidationErrorContextSmsPolicyViolation struct {
	Reason string
}

type CodeSentError struct {
	*schema.ValidationError
}

func (e CodeSentError) Error() string {
	return e.ValidationError.Error()
}

func (e CodeSentError) Unwrap() error {
	return e.ValidationError
}

func (e CodeSentError) StatusCode() int {
	return http.StatusOK
}

func NewSmsCodeSentError() error {
	return CodeSentError{
		ValidationError: &schema.ValidationError{
			ValidationError: &jsonschema.ValidationError{
				Message:     `access code has been sent`,
				InstancePtr: "#/",
				Context:     &ValidationErrorContextSmsPolicyViolation{},
			},
			Messages: new(text.Messages).Add(text.NewErrorSmsCodeSent()),
		}}
}

func (r *ValidationErrorContextSmsPolicyViolation) AddContext(_, _ string) {}

func (r *ValidationErrorContextSmsPolicyViolation) FinishInstanceContext() {}

func NewInvalidSmsCodeError() error {
	return schema.ValidationError{
		ValidationError: &jsonschema.ValidationError{
			Message:     `the provided code is invalid, check for spelling mistakes in the code or phone number`,
			InstancePtr: "#/",
			Context:     &ValidationErrorContextSmsPolicyViolation{},
		},
		Messages: new(text.Messages).Add(text.NewErrorValidationInvalidSmsCode()),
	}
}
