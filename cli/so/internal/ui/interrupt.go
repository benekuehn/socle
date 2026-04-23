package ui

import (
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/AlecAivazis/survey/v2/terminal"
)

// ErrPromptInterrupted indicates the user cancelled an interactive prompt.
var ErrPromptInterrupted = errors.New("prompt interrupted")

// HandleSurveyInterrupt checks for the specific survey interrupt error (Ctrl+C)
// and returns a sentinel error, otherwise returns the original error wrapped.
func HandleSurveyInterrupt(err error, message string) error {
	if errors.Is(err, terminal.InterruptErr) {
		_, _ = fmt.Fprintln(os.Stderr, message)
		return ErrPromptInterrupted
	}
	if errors.Is(err, io.EOF) {
		return fmt.Errorf("prompt failed: %w (received io.EOF, potentially non-interactive environment?)", err)
	}
	return fmt.Errorf("prompt failed: %w", err)
}
