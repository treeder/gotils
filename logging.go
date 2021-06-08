package gotils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
)

type contextKey string

// RequestIDContextKey is the name of the key used to store the request ID into the context
const (
	RequestIDContextKey = contextKey("request_id")
	LoggerContextKey    = contextKey("logger")
	errContext          = contextKey("errContext")
)

// Printfer just Printf
type Printfer interface {
	Printf(format string, v ...interface{})
}

// Printer common interface
type Printer interface {
	Print(v ...interface{})
	Println(v ...interface{})
	Printfer
}

// Wrapperr is an interface for Errorf
// see what I did there?
type Wrapperr interface {
	// creates an error and wraps it up
	Errorf(format string, a ...interface{}) error
	Printfer
	// wraps the error that's passed in
	Error(err error) error
}

// Fielder methods for adding structured fields
type Fielder interface {
	// F adds structured key/value pairs which will show up nicely in Cloud Logging.
	// Typically use this on the same line as your Printx()
	F(string, interface{}) Line
	// With clones (unlike F), then adds structured key/value pairs which will show up nicely in Cloud Logging.
	// Use this one if you plan on passing this along to other functions or setting global fields.
	With(string, interface{}) Line
}

// Leveler methods to set levels on loggers
type Leveler interface {
	// Debug returns a new logger with Debug severity
	Debug() Line
	// Info returns a new logger with INFO severity
	Info() Line
	// Error returns a new logger with ERROR severity
	Error() Line
}

// Line is the main interface returned from most functions
type Line interface {
	Fielder
	Printer
	Leveler
}

// Stacked is a wrapper for an error with a stack
type Stacked interface {
	Stack() []runtime.Frame
}

// Fielded is implemented if an object has fields on it
type Fielded interface {
	Fields() map[string]interface{}
}

// FullStacked has all the goodies in it
type FullStacked interface {
	error
	Fielded
	Stacked
	// maybe have a trace field in here? or the Fields can have a special trace key?
}

type stackedWrapper struct {
	err    error
	fields map[string]interface{}
	stack  []runtime.Frame
}

func (e *stackedWrapper) Error() string                  { return e.err.Error() }
func (e *stackedWrapper) Unwrap() error                  { return e.err }
func (e *stackedWrapper) Stack() []runtime.Frame         { return e.stack }
func (e *stackedWrapper) Fields() map[string]interface{} { return e.fields }

// With clones the error, then adds structured key/value pairs.
// Use this one if you plan on passing this along to other functions or setting global fields.
// todo: Provide similar function with user defined context key
func With(ctx context.Context, key string, value interface{}) context.Context {
	fields, ok := ctx.Value(errContext).(map[string]interface{})
	if !ok {
		fields = map[string]interface{}{}
	} else {
		// clone it
		fields2 := map[string]interface{}{}
		for k, v := range fields {
			fields2[k] = v
		}
		fields = fields2
	}
	fields[key] = value
	ctx = context.WithValue(ctx, errContext, fields)
	return ctx
}

// C use this to get an object back that has the regular Errorf signature.
func C(ctx context.Context) Wrapperr {
	return &wrapperr{ctx}
}

type wrapperr struct {
	ctx context.Context
}

func (w *wrapperr) Errorf(format string, a ...interface{}) error {
	return Errorf(w.ctx, format, a...)
}
func (w *wrapperr) Printf(format string, a ...interface{}) {
	Printf(w.ctx, format, a...)
}
func (w *wrapperr) Error(err error) error {
	return Error(w.ctx, err)
}

// Fields returns all the fields added via With(...)
func Fields(ctx context.Context) map[string]interface{} {
	fields := ctx.Value(errContext)
	if fields != nil {
		return fields.(map[string]interface{})
	}
	return nil
}

// Errorf just like fmt.Errorf, but takes a stack trace (if none exists already)
func Errorf(ctx context.Context, format string, a ...interface{}) FullStacked {
	e2 := fmt.Errorf(format, a...)
	// only take stacktrace if not already Stacked
	for _, x := range a {
		switch y := x.(type) {
		case error:
			var e *stackedWrapper
			if errors.As(y, &e) {
				// This was already called before, so we don't want to get a new stack trace or change the existing fields
				// Make a new wrapper so we don't lose any other errors in the chain
				e3 := &stackedWrapper{
					err:    e2,
					fields: e.fields,
					stack:  e.stack,
				}
				// add any new fields that may have been added
				fields, ok := ctx.Value(errContext).(map[string]interface{})
				if ok {
					for k, v := range fields {
						if e3.fields[k] == nil {
							e3.fields[k] = v
						}
					}
				}
				return e3
			}
		}
	}
	return Error(ctx, e2)
}

func Error(ctx context.Context, e2 error) FullStacked {
	var e *stackedWrapper
	if errors.As(e2, &e) {
		// This was already called before, so we don't want to get a new stack trace or change the existing fields
		// Make a new wrapper so we don't lose any other errors in the chain
		e3 := &stackedWrapper{
			err:    e2,
			fields: e.fields,
			stack:  e.stack,
		}
		// add any new fields that may have been added
		fields, ok := ctx.Value(errContext).(map[string]interface{})
		if ok {
			for k, v := range fields {
				if e3.fields[k] == nil {
					e3.fields[k] = v
				}
			}
		}
		return e3
	}
	fields, ok := ctx.Value(errContext).(map[string]interface{})
	if !ok {
		fields = map[string]interface{}{}
	}
	return &stackedWrapper{
		err:    e2,
		fields: fields,
		stack:  takeStacktrace(),
	}
}

func takeStacktrace() []runtime.Frame {
	pc := make([]uintptr, 25)
	_ = runtime.Callers(2, pc)
	frames := runtime.CallersFrames(pc)
	frames2 := []runtime.Frame{}
	for frame, more := frames.Next(); more; frame, more = frames.Next() {
		if shouldSkip(frame.Function) {
			continue
		}
		frames2 = append(frames2, frame)
	}
	return frames2
}

// Printf prints a message along with contextual data
func Printf(ctx context.Context, format string, a ...interface{}) {
	msg := fmt.Sprintf(format, a...)
	for _, x := range a {
		switch y := x.(type) {
		case error:
			var e *stackedWrapper
			if errors.As(y, &e) {
				// This was already called before, so we don't want to get a new stack trace or change the existing fields
				// Make a new wrapper so we don't lose any other errors in the chain
				fmt.Print(str(msg, e.fields, e.stack))
				return
			}
		}
	}
	fields, ok := ctx.Value(errContext).(map[string]interface{})
	if !ok {
		fields = map[string]interface{}{}
	}
	fmt.Print(str(msg, fields, nil))
}

func str(msg string, fields map[string]interface{}, stack []runtime.Frame) string {
	buffer := bytes.Buffer{}
	// todo: log lib should add severity, ie: ERROR
	buffer.WriteString(msg)
	buffer.WriteRune('\n')

	if fields != nil && len(fields) > 0 {
		buffer.WriteRune('\t')
		i := 0
		for k, v := range fields {
			buffer.WriteString(fmt.Sprintf("%v=%v", k, v))
			if i < len(fields)-1 {
				buffer.WriteString(", ")
			}
		}
		buffer.WriteRune('\n')
	}
	buffer.WriteRune('\n')
	buffer.WriteString("goroutine 1 [running]:\n")
	for i, frame := range stack {
		if i != 0 {
			buffer.WriteRune('\n')
		}
		buffer.WriteString(frame.Function)
		buffer.WriteRune('(')
		buffer.WriteString(fmt.Sprintf("%v", frame.PC))
		buffer.WriteRune(')')
		buffer.WriteRune('\n')
		buffer.WriteRune('\t')
		buffer.WriteString(frame.File)
		buffer.WriteRune(':')
		buffer.WriteString(strconv.Itoa(frame.Line))
		i++
	}
	return buffer.String()
}

// ErrString prints an error to console.
// See https://github.com/treeder/gcputils for example
func ErrString(err error) string {
	var e *stackedWrapper
	if !errors.As(err, &e) {
		return err.Error()
	}

	return str(err.Error(), e.Fields(), e.Stack())

}

func shouldSkip(s string) bool {
	// fmt.Println("should skip: ", s)
	if strings.HasPrefix(s, "github.com/treeder/gotils") {
		return true
	}
	return false
}
