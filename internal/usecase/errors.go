package usecase

import "fmt"

type ErrorCode string

const (
	CodeInvalidArgument    ErrorCode = "invalid_argument"
	CodeInvalidCredentials ErrorCode = "invalid_credentials"
	CodeInvalidSignature   ErrorCode = "invalid_signature"
	CodeDeprecatedSignAlgo ErrorCode = "deprecated_signature_algorithm"
	CodeStaleTimestamp     ErrorCode = "stale_timestamp"
	CodeConfiguration      ErrorCode = "configuration_error"
	CodeMonitorOffline     ErrorCode = "monitor_offline"
	CodeDuplicateOrder     ErrorCode = "duplicate_order"
	CodeOverloaded         ErrorCode = "overloaded"
	CodeNotFound           ErrorCode = "not_found"
	CodeInvalidState       ErrorCode = "invalid_state"
	CodeConflict           ErrorCode = "conflict"
	CodeDependency         ErrorCode = "dependency_error"
)

type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	if e.Cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.Cause)
}

func (e *Error) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Cause
}

func fail(code ErrorCode, message string) error {
	return &Error{Code: code, Message: message}
}

func wrap(code ErrorCode, message string, cause error) error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func ErrorCodeOf(err error) (ErrorCode, bool) {
	value, ok := err.(*Error)
	if !ok || value == nil {
		return "", false
	}
	return value.Code, true
}
