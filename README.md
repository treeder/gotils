# gotils

This is where I experiment with Go stuff.

## Logging

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

This is how it works:

```go
// add context fields
ctx = gotils.With(ctx, "abc", 123)
// We got an error from somewhere, so let's wrap it and capture the current context with stacktrace
return gotils.C(ctx).Errorf("uh oh: %w", err)
// return it, then next level adds message context, like normal
return gotils.C(ctx).Errorf("doh: %v", err)
// and so on until we are ready to return something to the user
gcputils.Print(err) // write to google cloud logging in proper structured format
fmt.Print(gotils.ErrString(err)) // write to user
```
