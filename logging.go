package gotils

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"runtime"
	"strconv"
	"strings"

	"github.com/rs/xid"
	"go.uber.org/zap"
)

type contextKey string

// RequestIDContextKey is the name of the key used to store the request ID into the context
const (
	RequestIDContextKey = contextKey("request_id")
	LoggerContextKey    = contextKey("logger")
	errContext          = contextKey("errContext")
)

/**********************
Note: all the zap related stuff is deprecated and will be removed at some point.
*****************/

// WithLogger call this when you first startup your app
func WithLogger(ctx context.Context, l *zap.Logger) context.Context {
	return context.WithValue(ctx, LoggerContextKey, l)
}

// Logger get the logger
func Logger(ctx context.Context) *zap.Logger {
	var logger *zap.Logger
	loggerInterface := ctx.Value(LoggerContextKey)
	if loggerInterface != nil {
		if lgr, ok := loggerInterface.(*zap.Logger); ok {
			logger = lgr
		}
	}
	if logger == nil {
		// add a default one to context and return it
		logger, err := zap.NewProduction()
		if err != nil {
			log.Fatalf("can't initialize zap logger: %v", err)
		}
		return logger
	}
	return logger
}

// L shortcut for Logger
func L(ctx context.Context) *zap.Logger {
	return Logger(ctx)
}

// WithRequestID stores a request ID into the context
func WithRequestID(ctx context.Context) context.Context {
	guid := xid.New()
	gs := guid.String()
	ctx = context.WithValue(ctx, contextKey(RequestIDContextKey), gs)
	ctx = AddFields(ctx, zap.String("request_id", gs))
	return ctx
}

// AddFields adds fields to the context logger
func AddFields(ctx context.Context, fields ...zap.Field) context.Context {
	l := Logger(ctx)
	l = l.With(fields...)
	ctx = WithLogger(ctx, l)
	return ctx
}

/********************
END zap related stuff
*********************/

// Printer common interface
type Printer interface {
	Print(v ...interface{})
	Println(v ...interface{})
	Printf(format string, v ...interface{})
}

// Wrapperr is an interface for Errorf
// see what I did there?
type Wrapperr interface {
	Errorf(format string, a ...interface{}) error
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

// Fields returns all the fields added via With(...)
func Fields(ctx context.Context) map[string]interface{} {
	fields := ctx.Value(errContext)
	if fields != nil {
		return fields.(map[string]interface{})
	}
	return nil
}

// Errorf just like fmt.Errorf, but takes a stack trace (if none exists already)
func Errorf(ctx context.Context, format string, a ...interface{}) error {
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

// ErrString temporary for how to print the stack, this should be in the logging lib.
// See https://github.com/treeder/gcputils for example
func ErrString(err error) string {
	buffer := bytes.Buffer{}
	// todo: log lib should add severity, ie: ERROR
	buffer.WriteString(err.Error())
	buffer.WriteRune('\n')

	var e *stackedWrapper
	if !errors.As(err, &e) {
		return err.Error()
	}

	if e.Fields() != nil && len(e.Fields()) > 0 {
		buffer.WriteRune('\t')
		i := 0
		for k, v := range e.Fields() {
			buffer.WriteString(fmt.Sprintf("%v=%v", k, v))
			if i < len(e.Fields())-1 {
				buffer.WriteString(", ")
			}
		}
		buffer.WriteRune('\n')
	}
	buffer.WriteRune('\n')
	buffer.WriteString("goroutine 1 [running]:\n")
	for i, frame := range e.Stack() {
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

func shouldSkip(s string) bool {
	// fmt.Println("should skip: ", s)
	if strings.HasPrefix(s, "github.com/treeder/gotils") {
		return true
	}
	return false
}
