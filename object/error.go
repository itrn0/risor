package object

import (
	"context"
	"fmt"

	"github.com/itrn0/risor/errz"
	"github.com/itrn0/risor/op"
)

// Error wraps a Go error interface and implements Object.
type Error struct {
	*base
	err    error
	raised bool
}

func (e *Error) Type() Type {
	return ERROR
}

func (e *Error) Inspect() string {
	return fmt.Sprintf("error(%q)", e.err.Error())
}

func (e *Error) String() string {
	return e.err.Error()
}

func (e *Error) Value() error {
	return e.err
}

func (e *Error) Interface() interface{} {
	return e.err
}

func (e *Error) Compare(other Object) (int, error) {
	otherErr, ok := other.(*Error)
	if !ok {
		return 0, errz.TypeErrorf("type error: unable to compare error and %s", other.Type())
	}
	thisMsg := e.Message().Value()
	otherMsg := otherErr.Message().Value()
	if thisMsg == otherMsg && e.raised == otherErr.raised {
		return 0, nil
	}
	if thisMsg > otherMsg {
		return 1, nil
	}
	if thisMsg < otherMsg {
		return -1, nil
	}
	if e.raised && !otherErr.raised {
		return 1, nil
	}
	if !e.raised && otherErr.raised {
		return -1, nil
	}
	return 0, nil
}

func (e *Error) Equals(other Object) Object {
	switch other := other.(type) {
	case *Error:
		if e.Message().Value() == other.Message().Value() && e.raised == other.raised {
			return True
		}
		return False
	default:
		return False
	}
}

func (e *Error) GetAttr(name string) (Object, bool) {
	switch name {
	case "error":
		return NewBuiltin("error", func(ctx context.Context, args ...Object) Object {
			return e.Message()
		}), true
	case "message":
		return NewBuiltin("message", func(ctx context.Context, args ...Object) Object {
			return e.Message()
		}), true
	default:
		return nil, false
	}
}

func (e *Error) Message() *String {
	return NewString(e.err.Error())
}

func (e *Error) WithRaised(value bool) *Error {
	e.raised = value
	return e
}

func (e *Error) IsRaised() bool {
	return e.raised
}

func (e *Error) Error() string {
	return e.err.Error()
}

func (e *Error) Unwrap() error {
	return e.err
}

func (e *Error) RunOperation(opType op.BinaryOpType, right Object) Object {
	return TypeErrorf("type error: unsupported operation for error: %v", opType)
}

func Errorf(format string, a ...interface{}) *Error {
	var args []interface{}
	for _, arg := range a {
		if obj, ok := arg.(Object); ok {
			args = append(args, obj.Interface())
		} else {
			args = append(args, arg)
		}
	}
	return &Error{err: fmt.Errorf(format, args...), raised: true}
}

func (e *Error) MarshalJSON() ([]byte, error) {
	return nil, errz.TypeErrorf("type error: unable to marshal error")
}

func NewError(err error) *Error {
	switch err := err.(type) {
	case *Error: // unwrap to get the inner error, to avoid unhelpful nesting
		return &Error{err: err.Unwrap(), raised: true}
	default:
		return &Error{err: err, raised: true}
	}
}

func IsError(obj Object) bool {
	if obj != nil {
		return obj.Type() == ERROR
	}
	return false
}
