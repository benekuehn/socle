package ui

import (
	"fmt"
	"os"
)

// HandleSurveyInterrupt checks for the specific survey interrupt error (Ctrl+C)
// and exits gracefully with a message, otherwise returns the original error wrapped.
func HandleSurveyInterrupt(err error, message string) error {
	if err.Error() == "interrupt" {
		fmt.Println(message)
		os.Exit(0) // Clean exit
	}
	return fmt.Errorf("prompt failed: %w", err)
}
