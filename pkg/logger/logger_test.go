package logger

import (
	"context"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func TestContextWithCorrelationID(t *testing.T) {
	ctx := ContextWithCorrelationID(context.Background(), "test-id")
	if got := CorrelationIDFromContext(ctx); got != "test-id" {
		t.Fatalf("expected correlation ID %q, got %q", "test-id", got)
	}
}

func TestWithContextAddsCorrelationIDField(t *testing.T) {
	core, recorded := observer.New(zapcore.InfoLevel)
	original := log
	log = zap.New(core)
	defer func() { log = original }()

	ctx := ContextWithCorrelationID(context.Background(), "context-id")

	WithContext(ctx).Info("test message")

	entries := recorded.All()
	if len(entries) != 1 {
		t.Fatalf("expected 1 log entry, got %d", len(entries))
	}

	correlationID, ok := entries[0].ContextMap()["correlation_id"]
	if !ok {
		t.Fatalf("expected correlation_id field to be present")
	}

	if correlationID != "context-id" {
		t.Fatalf("expected correlation_id %q, got %v", "context-id", correlationID)
	}
}
