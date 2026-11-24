//go:build !windows

package cmd

import "errors"

// cmd.exe exists only on Windows; elsewhere there is no AutoRun to read.
func readAutoRun() (string, error) { return "", nil }

func writeAutoRun(string) error { return errors.New("cmd's AutoRun exists only on Windows") }
