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
	// Internal error that shouldn't be displayed to the user
	Internal(err error) Wrapperr
}

// Leveler methods to set levels on loggers
type Leveler interface {
	// Debug returns a new logger with Debug severity
	Debug() Printer
	// Info returns a new logger with INFO severity
	Info() Printer
	// Error returns a new logger with ERROR severity
	Error() Printer
}

// // Line is the main interface returned from most functions
// type Line interface {
// 	// Fielder
// 	// Printer
// 	Leveler
// }

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

type internalError struct {
	err error
	msg string
}

func (e *internalError) Error() string { return e.err.Error() }
func (e *internalError) Unwrap() error { return e.err }

type stackedWrapper struct {
	err    error
	fields map[string]interface{}
	stack  []runtime.Frame
}

func (e *stackedWrapper) Error() string                  { return e.err.Error() }
func (e *stackedWrapper) Unwrap() error                  { return e.err }
func (e *stackedWrapper) Stack() []runtime.Frame         { return e.stack }
func (e *stackedWrapper) Fields() map[string]interface{} { return e.fields }

type Loggable interface {
	Logf(ctx context.Context, severity, format string, a ...interface{})
	Log(ctx context.Context, severity string, a ...interface{})
}

var (
	pf       Printfer
	loggable Loggable
)

// SetPrintfer to let this library print errors to your logging library
//
// Deprecated: Use SetLoggable
func SetPrintfer(p Printfer) {
	pf = p
}

// SetLoggable is where you can set where the logs will go
func SetLoggable(l Loggable) {
	loggable = l
}

// LogBeta is the general function for all logging.
// It will change from LogBeta to something better when I'm comfortable with this.
// https://github.com/treeder/gotils/issues/5
// This should be Logf
// Deprecated: use Logf
func LogBeta(ctx context.Context, severity, format string, a ...interface{}) {
	Logf(ctx, severity, format, a...)
}

// LogBeta2 this should be Log
// Deprecated: use Log
func LogBeta2(ctx context.Context, severity string, a ...interface{}) {
	Log(ctx, severity, a...)
}

// Logf the Printf style
func Logf(ctx context.Context, severity, format string, a ...interface{}) {
	if loggable == nil {
		// then just default to console
		Printf(ctx, format, a...)
		return
	}
	loggable.Logf(ctx, severity, format, a...)
}

// Log the Print/Println style
func Log(ctx context.Context, severity string, a ...interface{}) {
	s := fmt.Sprintln(a...)
	// for _, v := range a {
	// 	s += fmt.Sprintf("%v", v)
	// }
	Logf(ctx, severity, s)
}

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

func CopyCtxWithoutCancel(ctx context.Context) context.Context {
	ret := context.Background()
	fields, ok := ctx.Value(errContext).(map[string]interface{})
	if !ok {
		return ret
	}
	// clone it
	fields2 := map[string]interface{}{}
	for k, v := range fields {
		fields2[k] = v
	}
	fields = fields2
	ret = context.WithValue(ret, errContext, fields)
	return ret
}

// NewLine returns an object that deals with logging
func NewLine(ctx context.Context) Leveler {
	return &line{ctx: ctx}
}

type line struct {
	ctx context.Context
	sev string
	// fields map[string]interface{}
	// trace  string
}

func (l *line) Debug() Printer {
	// l2 := l.clone()
	l.sev = "debug"
	return l
}

func (l *line) Info() Printer {
	// l2 := l.clone()
	l.sev = "info"
	return l
}
func (l *line) Error() Printer {
	// l2 := l.clone()
	l.sev = "error"
	return l
}

// Printf prints to the appropriate destination
// Arguments are handled in the manner of fmt.Printf.
func (l *line) Printf(format string, v ...interface{}) {
	LogBeta(l.ctx, l.sev, format, v...)
}

// Println prints to the appropriate destination
// Arguments are handled in the manner of fmt.Println.
func (l *line) Println(v ...interface{}) {
	l.Print(v...)
}

// Print prints to the appropriate destination
// Arguments are handled in the manner of fmt.Print.
func (l *line) Print(v ...interface{}) {
	LogBeta2(l.ctx, l.sev, v...)
}

// L returns an object that deals with logging
func L(ctx context.Context) Leveler {
	return NewLine(ctx)
}

// C use this to get an object that deals with errors.
func C(ctx context.Context) Wrapperr {
	return &wrapperr{ctx: ctx}
}

type wrapperr struct {
	ctx           context.Context
	internalError error
}

func (w *wrapperr) Errorf(format string, a ...interface{}) error {
	if w.internalError != nil {
		// wrap the internal one first
		return &internalError{err: w.internalError, msg: fmt.Sprintf(format, a...)}
	}
	return Errorf(w.ctx, format, a...)
}
func (w *wrapperr) Printf(format string, a ...interface{}) {
	Printf(w.ctx, format, a...)
}
func (w *wrapperr) Error(err error) error {
	return Error(w.ctx, err)
}
func (w *wrapperr) Internal(err error) Wrapperr {
	w.internalError = Error(w.ctx, err) // wrap it and stack it
	return w
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
				// This was already called before, so we don't want to get a new stack trace or change the existing fields,
				// but make new wrapper so we don't lose any other errors in the chain
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
	if msg[len(msg)-1] != '\n' {
		buffer.WriteRune('\n')
	}

	if len(fields) > 0 {
		buffer.WriteRune('\t')
		i := 0
		for k, v := range fields {
			buffer.WriteString(fmt.Sprintf("%v: %v", k, v))
			if i < len(fields)-1 {
				buffer.WriteString("\n\t")
			}
			i++
		}
		buffer.WriteRune('\n')
	}
	if stack != nil {
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
	}
	return buffer.String()
}

// ErrString returns a string representation of error including stacktrace, etc
// See https://github.com/treeder/gcputils for example
func ErrString(err error) string {
	var e *stackedWrapper
	if !errors.As(err, &e) {
		return err.Error()
	}

	return str(err.Error(), e.Fields(), e.Stack())

}

func shouldSkip(s string) bool {
	// fmt.Printf("should skip? %v\n", s)
	return strings.HasPrefix(strings.TrimSpace(s), "github.com/treeder/gotils")
}
