package wizard

import (
	"errors"

	"charm.land/huh/v2"
)

// huhPrompter is the real terminal Prompter backed by charmbracelet/huh. Each method
// runs a single blocking field.
type huhPrompter struct{}

// NewHuhPrompter returns a Prompter that drives an interactive huh terminal UI.
func NewHuhPrompter() Prompter { return huhPrompter{} }

// mapAbort converts huh's user-cancel sentinel into our ErrAborted.
func mapAbort(err error) error {
	if errors.Is(err, huh.ErrUserAborted) {
		return ErrAborted
	}
	return err
}

func (huhPrompter) Input(label, placeholder string, validate func(string) error) (string, error) {
	var v string
	in := huh.NewInput().Title(label).Placeholder(placeholder).Value(&v)
	if validate != nil {
		in = in.Validate(validate)
	}
	if err := in.Run(); err != nil {
		return "", mapAbort(err)
	}
	return v, nil
}

func (huhPrompter) Password(label string) (string, error) {
	var v string
	if err := huh.NewInput().Title(label).EchoMode(huh.EchoModePassword).Value(&v).Run(); err != nil {
		return "", mapAbort(err)
	}
	return v, nil
}

func (huhPrompter) Select(label string, options []string) (string, error) {
	var v string
	if err := huh.NewSelect[string]().Title(label).Options(huh.NewOptions(options...)...).Value(&v).Run(); err != nil {
		return "", mapAbort(err)
	}
	return v, nil
}

func (huhPrompter) Confirm(label string, defaultVal bool) (bool, error) {
	v := defaultVal
	if err := huh.NewConfirm().Title(label).Value(&v).Run(); err != nil {
		return false, mapAbort(err)
	}
	return v, nil
}
