Golang logger that aims to provide a simple but complete interface for a logger.

# Global or local?

## Declare a global logger

```go
// Set logger
astilog.SetLogger(astilog.New(astilog.Configuration{}))

// Use logger
astilog.Info("This is a log message")
```

## Declare a local logger

```go
// Create logger
l := astilog.New(astilog.Configuration{})

// Use logger
l.Info("This is a log message")
```