// Package wizard implements the interactive `md2wiki init` configuration flow.
package wizard

import (
	"errors"
	"strings"
)

// ErrAborted is returned when the user cancels the wizard (e.g. Ctrl-C). The CLI
// treats it as a clean, non-error exit. huhPrompter maps huh.ErrUserAborted to this.
var ErrAborted = errors.New("cancelled")

// Choice is a selectable option: Value is stored/returned, Desc is a short hint shown
// to the user (huh displays it under the field and updates it as the cursor moves).
type Choice struct {
	Value string
	Desc  string
}

// Prompter asks the user one value at a time. huhPrompter is the real implementation;
// tests use a fake. validate (when non-nil) is applied to Input values before returning.
type Prompter interface {
	Input(label, placeholder string, validate func(string) error) (string, error)
	Password(label string) (string, error)
	Select(label string, choices []Choice) (string, error)
	Confirm(label string, defaultVal bool) (bool, error)
}

func validateNonEmpty(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("value is required")
	}
	return nil
}

func validateURL(s string) error {
	if err := validateNonEmpty(s); err != nil {
		return err
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return errors.New("must start with http:// or https://")
	}
	return nil
}
