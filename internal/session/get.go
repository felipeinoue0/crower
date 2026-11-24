package session

import (
	"github.com/felipeinoue0/crower/internal/domain/operation"
	"github.com/felipeinoue0/crower/internal/domain/timeline"
)

// Get last operation
func (s *Session) GetLast(state operation.Operation) *timeline.Content {
	return s.History.GetLastOperation(state)
}
