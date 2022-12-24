package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/treeder/gotils/v2"
)

func main() {
	ctx := context.Background()
	ctx = gotils.With(ctx, "abc", 123)
	// we got an error from somewhere:
	err := errors.New("something bad")
	// fmt.Println(gotils.ErrString(err))
	err = gotils.Errorf(ctx, "uh oh: %v", err)
	// fmt.Println(gotils.ErrString(err))
	// fmt.Println("fields:", gotils.Fields(ctx))
	err = gotils.C(ctx).Errorf("doh: %v", err)
	fmt.Println(gotils.ErrString(err))

	err = gotils.C(ctx).SetCode(http.StatusBadRequest).Message("this is for the user: bad input yo!").Errorf("bad input: %w", err)
	fmt.Println(gotils.ErrString(err))
	DumpError(err)

	// err = gotils.C(ctx).SetCode(http.StatusBadRequest). // Errorf("bad input: %w", err)
	// fmt.Println(gotils.ErrString(err))
	var coded gotils.Coded
	if errors.As(err, &coded) {
		fmt.Println("CODE:", coded.Code())
	}
	var um gotils.UserMessage
	if errors.As(err, &um) {
		fmt.Println("USER MESSAGE:", um.Message())
		fmt.Println("USER MESSAGE as ERR:", um.Error())
	}

}

func DumpError(err error) {
	if err == nil {
		return
	}
	fmt.Println(reflect.TypeOf(err), err)
	DumpError(errors.Unwrap(err))
}

// func main() {
// 	ctx := context.Background()
// 	err := errors.New("error 1")
// 	fmt.Println(gotils.ErrString(err))
// 	err = gotils.Errorf(ctx, "error 2: %v", err)
// 	fmt.Println(gotils.ErrString(err))
// 	ctx = gotils.With(ctx, "abc", 123)
// 	fmt.Println("fields:", gotils.Fields(ctx))
// 	err = gotils.C(ctx).Errorf("error3: %v", err)
// 	fmt.Println(gotils.ErrString(err))

// 	// show a regular stack trace too
// 	yo()
// }

// func yo() {
// 	b := make([]byte, 1<<20)
// 	runtime.Stack(b, false)
// 	fmt.Println("Regular stack")
// 	fmt.Println(string(b))
// }
