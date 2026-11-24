package notes

import (
	"github.com/felipeinoue0/crower/internal/domain/operation"
)

// Revert renders the history note for a revert.
func Revert(args []string) string {
	return New().
		AddMainOperation(operation.Revert).
		AddCrowerExec("revert", args).
		Build()
}
