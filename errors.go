package exturl

import (
	"fmt"
)

//
// NotFound
//

type NotFound struct {
	Message string
}

func NewNotFound(message string) *NotFound {
	return &NotFound{message}
}

func NewNotFoundf(format string, arg ...any) *NotFound {
	return NewNotFound(fmt.Sprintf(format, arg...))
}

// error interface
func (self *NotFound) Error() string {
	return self.Message
}

func IsNotFound(err error) bool {
	_, ok := err.(*NotFound)
	return ok
}

//
// NotImplemented
//

type NotImplemented struct {
	Message string
}

func NewNotImplemented(message string) *NotImplemented {
	return &NotImplemented{message}
}

func NewNotImplementedf(format string, arg ...any) *NotImplemented {
	return NewNotImplemented(fmt.Sprintf(format, arg...))
}

// error interface
func (self *NotImplemented) Error() string {
	return self.Message
}

func IsNotImplemented(err error) bool {
	_, ok := err.(*NotImplemented)
	return ok
}
