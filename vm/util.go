package vm

import (
	"fmt"
	"log/slog"
	"reflect"
	"unsafe"

	"github.com/itrn0/risor/errz"
	"github.com/itrn0/risor/object"
)

const (
	ObjectIntSize        = int(unsafe.Sizeof(object.Int{}))
	ObjectFloatSize      = int(unsafe.Sizeof(object.Int{}))
	ObjectBoolSize       = int(unsafe.Sizeof(object.Bool{}))
	ObjectStringSize     = int(unsafe.Sizeof(object.String{}))
	ObjectNilSize        = int(unsafe.Sizeof(object.Nil))
	ObjectTimeSize       = int(unsafe.Sizeof(object.Time{}))
	ObjectChanSize       = int(unsafe.Sizeof(object.Chan{}))
	ObjectMapSize        = int(unsafe.Sizeof(object.Map{}))
	ObjectByteSliceSize  = int(unsafe.Sizeof(object.ByteSlice{}))
	ObjectFloatSliceSize = int(unsafe.Sizeof(object.FloatSlice{}))
	ObjectListSize       = int(unsafe.Sizeof(object.List{}))
	ObjectErrorSize      = int(unsafe.Sizeof(object.Error{}))
	ObjectModuleSize     = int(unsafe.Sizeof(object.Module{}))
	ObjectPartialSize    = int(unsafe.Sizeof(object.Partial{}))
	ObjectFunctionSize   = int(unsafe.Sizeof(object.Function{}))
	ObjectBuiltinSize    = int(unsafe.Sizeof(object.Builtin{}))
	ObjectIntIterSize    = int(unsafe.Sizeof(object.IntIter{}))
	ObjectListIterSize   = int(unsafe.Sizeof(object.ListIter{}))
	ObjectMapIterSize    = int(unsafe.Sizeof(object.MapIter{}))
	ObjectSetIterSize    = int(unsafe.Sizeof(object.SetIter{}))
	ObjectSliceIterSize  = int(unsafe.Sizeof(object.SliceIter{}))

	StringSize = int(unsafe.Sizeof(""))
	ArraySize  = int(unsafe.Sizeof([]any{}))
	IntSize    = int(unsafe.Sizeof(0))
	PtrSize    = int(unsafe.Sizeof(unsafe.Pointer(nil)))
)

func checkCallArgs(fn *object.Function, argc int) error {
	// Number of parameters in the function signature
	paramsCount := len(fn.Parameters())

	// Number of required args when the function is called (those without defaults)
	requiredArgsCount := fn.RequiredArgsCount()

	// Check if too many or too few arguments were passed
	if argc > paramsCount || argc < requiredArgsCount {
		msg := "args error: function"
		if name := fn.Name(); name != "" {
			msg = fmt.Sprintf("%s %q", msg, name)
		}
		switch paramsCount {
		case 0:
			msg = fmt.Sprintf("%s takes 0 arguments (%d given)", msg, argc)
		case 1:
			msg = fmt.Sprintf("%s takes 1 argument (%d given)", msg, argc)
		default:
			msg = fmt.Sprintf("%s takes %d arguments (%d given)", msg, paramsCount, argc)
		}
		return errz.ArgsErrorf(msg)
	}
	return nil
}

func varSize(value any) (int, error) {
	switch val := value.(type) {
	case nil:
		return 0, nil
	case *object.Bool, object.Bool:
		return ObjectBoolSize, nil
	case *object.Float, object.Float:
		return ObjectFloatSize, nil
	case *object.Int, object.Int:
		return ObjectIntSize, nil
	case *object.String:
		return ObjectStringSize + len(val.Value()), nil
	case object.String:
		return ObjectStringSize + len(val.Value()), nil
	case *object.ByteSlice:
		return ObjectByteSliceSize + len(val.Value()), nil
	case *object.FloatSlice:
		return ObjectFloatSliceSize + len(val.Value()), nil
	case *object.List:
		var size int
		for _, v := range val.Value() {
			subSize, err := varSize(v)
			if err != nil {
				return 0, err
			}
			size += subSize
		}
		return ObjectListSize + size, nil
	case *object.Map:
		var size int
		for k, v := range val.Value() {
			size += StringSize + len(k)
			subSize, err := varSize(v)
			if err != nil {
				return 0, err
			}
			size += subSize
		}
		return ObjectMapSize + size, nil
	case *object.Partial:
		size, err := varSize(val.Args())
		if err != nil {
			return 0, err
		}
		return ObjectPartialSize + size, nil
	case object.Error:
		var size int
		if val.Value() != nil {
			size += ObjectStringSize + len(val.Value().Error())
		}
		return ObjectErrorSize + size, nil
	case *object.Builtin:
		return ObjectBuiltinSize + len(val.Name()), nil
	case *object.Module:
		var size int
		name := val.Name()
		if name != nil {
			subSize, err := varSize(*name)
			if err != nil {
				return 0, err
			}
			size += subSize - ObjectStringSize
		}
		return int(unsafe.Sizeof(val)) + size, nil
	case *object.Function:
		var size int
		for _, param := range val.Parameters() {
			size += len(param)
		}
		for _, def := range val.Defaults() {
			subSize, err := varSize(def)
			if err != nil {
				return 0, err
			}
			size += subSize
		}
		size += len(val.Name())
		return ObjectFunctionSize + size, nil
	case *object.NilType:
		return ObjectNilSize, nil
	case []object.Object:
		var size int
		for _, v := range val {
			subSize, err := varSize(v)
			if err != nil {
				return 0, err
			}
			size += subSize
		}
		return ArraySize + size, nil
	case *object.IntIter:
		return ObjectIntIterSize, nil
	case *object.ListIter:
		return ObjectListIterSize, nil
	case *object.MapIter:
		return ObjectMapIterSize, nil
	case *object.SetIter:
		return ObjectSetIterSize, nil
	case *object.SliceIter:
		return ObjectSliceIterSize, nil
	default:
		// Для остальных типов fallback на рефлексию
		slog.Info(
			"unsupported variable type",
			"type", reflect.TypeOf(value),
		)
		return calculateWithReflection(value)
	}
}

func calculateWithReflection(value any) (int, error) {
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Ptr:
		if v.IsNil() {
			return 0, nil
		}
		return varSize(v.Elem().Interface())
	case reflect.Struct, reflect.Func, reflect.Chan:
		return int(unsafe.Sizeof(value)), nil
	default:
		return 0, fmt.Errorf("unsupported variable type: %T", value)
	}
}
