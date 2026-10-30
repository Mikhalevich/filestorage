package perror

import (
	"errors"
)

type Error struct {
	Type    Type
	Message string
}

func New(t Type, msg string) Error {
	return Error{
		Type:    t,
		Message: msg,
	}
}

func ParseError(err error) (Error, bool) {
	if perr, ok := errors.AsType[Error](err); ok {
		return perr, true
	}

	return New(TypeUnspecified, "unspecified error"), false
}

func IsType(err error, t Type) bool {
	if perr, ok := errors.AsType[Error](err); ok {
		if perr.Type == t {
			return true
		}
	}

	return false
}

func (e Error) Error() string {
	return e.Message
}

func NotFound(msg string) Error {
	return New(TypeNotFound, msg)
}

func InvalidArgument(msg string) Error {
	return New(TypeInvalidArgument, msg)
}

func AlreadyExists(msg string) Error {
	return New(TypeAlreadyExists, msg)
}

func Unauthorized(msg string) Error {
	return New(TypeUnauthorized, msg)
}
