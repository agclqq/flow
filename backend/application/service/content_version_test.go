package service

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"flow/domain/entity"
	"flow/domain/valueobject"
)

func TestPickContentVersion(t *testing.T) {
	draft, _ := entity.NewFlowTemplateVersion(1, 2)
	draft.Status = valueobject.TemplateStatusDraft

	published, _ := entity.NewFlowTemplateVersion(1, 1)
	published.Status = valueobject.TemplateStatusPublished

	t.Run("prefers draft", func(t *testing.T) {
		got, err := pickContentVersion([]*entity.FlowTemplateVersion{draft, published})
		assert.NoError(t, err)
		assert.Equal(t, draft, got)
	})

	t.Run("falls back to published", func(t *testing.T) {
		got, err := pickContentVersion([]*entity.FlowTemplateVersion{published})
		assert.NoError(t, err)
		assert.Equal(t, published, got)
	})

	t.Run("empty list", func(t *testing.T) {
		_, err := pickContentVersion(nil)
		assert.Error(t, err)
	})
}
