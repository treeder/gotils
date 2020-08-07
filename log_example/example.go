package main

import (
	"context"
	"errors"
	"fmt"
	"runtime"

	"github.com/treeder/gotils"
)

func main() {
	ctx := context.Background()
	err := errors.New("error 1")
	fmt.Println(gotils.ErrString(err))
	err = gotils.Errorf(ctx, "error 2: %v", err)
	fmt.Println(gotils.ErrString(err))
	ctx = gotils.With(ctx, "abc", 123)
	fmt.Println("fields:", gotils.Fields(ctx))
	err = gotils.C(ctx).Errorf("error3: %v", err)
	fmt.Println(gotils.ErrString(err))

	// show a regular stack trace too
	yo()
}

func yo() {
	b := make([]byte, 1<<20)
	runtime.Stack(b, false)
	fmt.Println("Regular stack")
	fmt.Println(string(b))
}
