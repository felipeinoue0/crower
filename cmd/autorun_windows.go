//go:build windows

package cmd

import (
	"errors"

	"github.com/felipeinoue0/crower/internal/adapters/shellsetup"

	"golang.org/x/sys/windows/registry"
)

// readAutoRun is cmd's AutoRun value for the current user, "" when unset.
func readAutoRun() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, shellsetup.AutoRunKey, registry.QUERY_VALUE)
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}

	if err != nil {
		return "", err
	}
	defer key.Close()

	value, _, err := key.GetStringValue(shellsetup.AutoRunValue)
	if errors.Is(err, registry.ErrNotExist) {
		return "", nil
	}

	return value, err
}

// writeAutoRun sets cmd's AutoRun value for the current user.
func writeAutoRun(value string) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, shellsetup.AutoRunKey, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.SetStringValue(shellsetup.AutoRunValue, value)
}
