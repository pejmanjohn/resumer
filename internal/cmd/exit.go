package cmd

import "errors"

const (
	ExitOK        = 0
	ExitUsage     = 2
	ExitDiscovery = 20
	ExitEmpty     = 21
	ExitCanceled  = 30
	ExitLaunch    = 40
)

type UsageError struct {
	Message string
}

func (e UsageError) Error() string {
	return e.Message
}

type CanceledError struct{}

func (e CanceledError) Error() string {
	return "selection canceled"
}

type LaunchError struct {
	Message string
}

func (e LaunchError) Error() string {
	return e.Message
}

func ExitCode(err error) int {
	if err == nil {
		return ExitOK
	}
	var usage UsageError
	if errors.As(err, &usage) {
		return ExitUsage
	}
	var canceled CanceledError
	if errors.As(err, &canceled) {
		return ExitCanceled
	}
	var launch LaunchError
	if errors.As(err, &launch) {
		return ExitLaunch
	}
	return ExitDiscovery
}
