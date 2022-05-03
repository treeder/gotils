# gotils

This is where I experiment with Go stuff.

[Documentation](https://pkg.go.dev/github.com/treeder/gotils/v2)

## Update Go Version

```sh
curl -LSs https://raw.githubusercontent.com/treeder/gotils/master/update.sh | bash
```

## Logging

[A blog post about this](https://betterprogramming.pub/fun-or-not-with-golang-errors-26b2b0e231c5)

Why? I was getting annoyed with things like this all over the place:

```go
if err != nil {
    log.Error("something bad", zap.Error(err)) // log something here with your favorite logging library
    return fmt.Errorf("something bad:", err)
}
```

Then as it goes back down the stack you don't know if you logged already or not so you keep logging and get duplicate messages. And you can't just pass the error back and log before you return the error because you lose all the context
that you added to the logging library via the structured fields and you lose the stacktrace. 

[Read more here](https://github.com/treeder/gotils/issues/2)

So I started thinking how to make this better by adding context to errors rather than to the logging library. I
am aware of the [pkg/errors](https://github.com/pkg/errors) lib, but it's in maintenance mode now and isn't compatible
with the new error features (ie: error wrapping) and it doesn't let you add fields.

So a few things I'm trying here:

1. Add context along the way
1. Collect stacktrace at the deepest point where the error happens (as close to the root error as possible)
1. Write error message, context fields and stacktrace in the appropriate format for your logging system

```go
// Add context:
ctx = gotils.With(ctx, "foo", "bar")
// Return errors with embedded stacktrace
return gotils.C(ctx).Errorf("something bad happened: %v", err)
// Then log it wherever you want:
if err != nil {
    // this writes to your provided logger or the console if no logger set in SetLoggable
    gotils.Logf(ctx, "error", "%v", err)
}
```

To log in a particular format or to send them to another service, set a `Loggable` instance: 

```go
gotils.SetLoggable(gcputils.NewLogger())
```

## HTTP Error Handler

To get use the above error context capturing and make HTTP error handling really clean and easy, try this.

First have your http endpoint functions return an error:

```go
func foo(w http.ResponseWriter, r *http.Request) error {
    // do stuff
    return err
}
```

Then wrap and add your http function like this:

```go
http.HandleFunc("/bar", gotils.ErrorHandler(foo))
```

Now it will handle errors and return nice JSON error responses without you having to do anything.

## HTTP Utils

```go
// GET JSON from API
v := &MyObject{}
err := gotils.GetJSON(url, v) // also GetString, GetBytes

// POST JSON to an API
err := gotils.PostJSON(url, v)

// And some response handling
gotils.WriteObject(w, 200, v) // also WriteMessage, WriteError
```

