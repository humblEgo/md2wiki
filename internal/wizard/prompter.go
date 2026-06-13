// Package wizard implements the interactive `md2wiki init` configuration flow.
package wizard

import (
	"errors"
	"strings"
)

// ErrAborted is returned when the user cancels the wizard (e.g. Ctrl-C). The CLI
// treats it as a clean, non-error exit. huhPrompter maps huh.ErrUserAborted to this.
var ErrAborted = errors.New("취소되었습니다")

// Prompter asks the user one value at a time. huhPrompter is the real implementation;
// tests use a fake. validate (when non-nil) is applied to Input values before returning.
type Prompter interface {
	Input(label, placeholder string, validate func(string) error) (string, error)
	Password(label string) (string, error)
	Select(label string, options []string) (string, error)
	Confirm(label string, defaultVal bool) (bool, error)
}

func validateNonEmpty(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("값을 입력하세요")
	}
	return nil
}

func validateURL(s string) error {
	if err := validateNonEmpty(s); err != nil {
		return err
	}
	if !strings.HasPrefix(s, "http://") && !strings.HasPrefix(s, "https://") {
		return errors.New("http:// 또는 https:// 로 시작해야 합니다")
	}
	return nil
}
