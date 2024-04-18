package signal

import (
	"context"
	"golang.org/x/sync/errgroup"
	"os"
	"os/signal"
)

var onlyOneSignalHandler = make(chan struct{})
var shutdownHandler chan os.Signal

// SetupStopSignalHandler registered for SIGTERM and SIGINT. A stop channel is returned
// which is closed on one of these signals. If a second signal is caught, the program
// is terminated with exit code 1.
// copied from https://github.com/kubernetes/apiserver/blob/master/pkg/server/signal.go
func SetupStopSignalHandler() <-chan struct{} {
	close(onlyOneSignalHandler) // panics when called twice

	shutdownHandler = make(chan os.Signal, 2)

	stop := make(chan struct{})
	signal.Notify(shutdownHandler, shutdownSignals...)
	go func() {
		<-shutdownHandler
		close(stop)
		<-shutdownHandler
		os.Exit(1) // second signal. Exit directly.
	}()

	return stop
}

// SetupStopSignalContext works similarly to SetupStopSignalHandler. It returns two objects.
// One is an errgroup.Group object (refer to https://godoc.org/golang.org/x/sync/errgroup to
// see how errgroup.Group works); aother is a channel that is closed when either a SIGTERM or
// a SIGINT signal is received or when one of the task that was executed by the Group is done.
func SetupStopSignalContext() (*errgroup.Group, <-chan struct{}) {
	group, ctx := errgroup.WithContext(Context(SetupStopSignalHandler()))
	return group, ctx.Done()
}

// Context returns a context that is cancelled upon the given signal
func Context(signal <-chan struct{}) context.Context {
	ret, cancel := context.WithCancel(context.Background())
	go func() {
		<-signal
		cancel()
	}()
	return ret
}
