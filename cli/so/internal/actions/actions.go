package actions

import "fmt"

// ErrExitSilently prevents Cobra from printing the error message.
type ErrExitSilently struct {
	ExitCode int
}

func (e ErrExitSilently) Error() string {
	// Provide a basic message for logging, but Cobra won't print this for ExitCode > 0
	return fmt.Sprintf("exiting silently with code %d", e.ExitCode)
}
