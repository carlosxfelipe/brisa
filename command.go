package brisa

import (
	"fmt"
	"log"
	"reflect"
	"sync"
	"unicode"

	"github.com/carlosxfelipe/brisa/core"
	"github.com/carlosxfelipe/brisa/validation"
)

var (
	loggedTagWarnings sync.Map
	emptyType         = reflect.TypeOf(Empty{})
	emptyStructType   = reflect.TypeOf(struct{}{})
	contextPtrType    = reflect.TypeOf((*core.Context)(nil))
	errorInterface    = reflect.TypeOf((*error)(nil)).Elem()
)

func isEmptyType(t reflect.Type) bool {
	return t == emptyType || t == emptyStructType
}

// CommandRegistrar is implemented by *App and *CommandGroup to allow registering typed REPR commands.
type CommandRegistrar interface {
	RegisterCommand(name string, handler HandlerFunc, doc CommandDoc) *Command
}

// CommandHandler defines the typed REPR function signature for Brisa IPC commands.
type CommandHandler[TReq any, TRes any] func(ctx *Context, req TReq) (TRes, error)

// Endpoint defines the REPR struct contract for a Brisa desktop command.
type Endpoint[TReq any, TRes any] interface {
	Handle(ctx *Context, req TReq) (TRes, error)
}

// Bind registers a typed REPR command handler (Request -> Endpoint -> Response) with automatic
// JSON binding, declarative struct validation, and schema introspection.
func Bind[TReq any, TRes any](r CommandRegistrar, name string, handler CommandHandler[TReq, TRes]) *Command {
	var zeroReq TReq
	var zeroRes TRes
	reqType := reflect.TypeOf(zeroReq)
	resType := reflect.TypeOf(zeroRes)

	if err := validation.CheckTags(reqType); err != nil {
		panic(fmt.Sprintf("brisa: invalid validate tag on command %q (%v): %v", name, reqType, err))
	}
	if !validation.SilenceTagWarnings {
		for _, w := range validation.CheckTagWarnings(reqType) {
			if _, loaded := loggedTagWarnings.LoadOrStore(w, true); !loaded {
				if validation.TagWarningLogger != nil {
					validation.TagWarningLogger(w)
				} else {
					log.Printf("[BRISA WARNING] %s", w)
				}
			}
		}
	}

	needBind := reqType != nil && !isEmptyType(reqType)

	wrapped := func(ctx *Context) error {
		var req TReq
		if needBind {
			if err := ctx.BindJSON(&req); err != nil {
				return err
			}
			if valErr := validation.ValidateStruct(ctx, &req); valErr != nil {
				return valErr
			}
		}

		res, err := handler(ctx, req)
		if err != nil {
			return err
		}
		if resType != nil && isEmptyType(resType) {
			return ctx.Reply(200, nil)
		}
		return ctx.Reply(200, res)
	}

	return r.RegisterCommand(name, wrapped, CommandDoc{
		Name:        name,
		RequestType: reqType,
		ReplyType:   resType,
	})
}

// BindEndpoint registers a struct-based REPR Endpoint for the given IPC command name.
func BindEndpoint[TReq any, TRes any](r CommandRegistrar, name string, endpoint Endpoint[TReq, TRes]) *Command {
	return Bind(r, name, endpoint.Handle)
}

// BindService reflects over all exported methods of service and registers each compatible method
// as a typed IPC command under "<namespace>.<camelCaseMethodName>".
//
// Supported method signatures:
//   - func(ctx *brisa.Context, req Req) (Res, error)
//   - func(req Req) (Res, error)
//   - func(ctx *brisa.Context) (Res, error)
//   - func() (Res, error)
func BindService(r CommandRegistrar, namespace string, service any) []*Command {
	if service == nil {
		panic("brisa: BindService called with nil service")
	}
	val := reflect.ValueOf(service)
	typ := val.Type()

	var registered []*Command
	for i := 0; i < typ.NumMethod(); i++ {
		m := typ.Method(i)
		if !m.IsExported() {
			continue
		}
		cmd, ok := bindReflectedMethod(r, namespace, val.Method(i), m)
		if ok {
			registered = append(registered, cmd)
		}
	}
	return registered
}

func bindReflectedMethod(r CommandRegistrar, namespace string, methodVal reflect.Value, method reflect.Method) (*Command, bool) {
	mType := methodVal.Type()
	numIn := mType.NumIn()
	numOut := mType.NumOut()

	if numOut != 2 || !mType.Out(1).Implements(errorInterface) {
		return nil, false
	}

	hasCtx := false
	var reqType reflect.Type

	switch numIn {
	case 0:
		// func() (Res, error)
	case 1:
		if mType.In(0) == contextPtrType {
			hasCtx = true
		} else {
			reqType = mType.In(0)
		}
	case 2:
		if mType.In(0) != contextPtrType {
			return nil, false
		}
		hasCtx = true
		reqType = mType.In(1)
	default:
		return nil, false
	}

	resType := mType.Out(0)
	cmdName := toCamelCase(method.Name)
	if namespace != "" {
		cmdName = namespace + "." + cmdName
	}

	if reqType != nil {
		if err := validation.CheckTags(reqType); err != nil {
			panic(fmt.Sprintf("brisa: invalid validate tag on service method %q (%v): %v", cmdName, reqType, err))
		}
	}

	needBind := reqType != nil && !isEmptyType(reqType)

	handler := func(ctx *Context) error {
		var args []reflect.Value
		if hasCtx {
			args = append(args, reflect.ValueOf(ctx))
		}
		if reqType != nil {
			isPtr := reqType.Kind() == reflect.Pointer
			baseType := reqType
			if isPtr {
				baseType = reqType.Elem()
			}
			reqPtr := reflect.New(baseType)
			if needBind {
				if err := ctx.BindJSON(reqPtr.Interface()); err != nil {
					return err
				}
				if valErr := validation.ValidateStruct(ctx, reqPtr.Interface()); valErr != nil {
					return valErr
				}
			}
			if isPtr {
				args = append(args, reqPtr)
			} else {
				args = append(args, reqPtr.Elem())
			}
		}

		out := methodVal.Call(args)
		if !out[1].IsNil() {
			return out[1].Interface().(error)
		}
		if isEmptyType(resType) {
			return ctx.Reply(200, nil)
		}
		return ctx.Reply(200, out[0].Interface())
	}

	cmd := r.RegisterCommand(cmdName, handler, CommandDoc{
		Name:        cmdName,
		Group:       namespace,
		RequestType: reqType,
		ReplyType:   resType,
	})
	return cmd, true
}

func toCamelCase(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	runes[0] = unicode.ToLower(runes[0])
	return string(runes)
}
