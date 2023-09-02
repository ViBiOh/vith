package vith

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (s Service) increaseMetric(ctx context.Context, source, kind, itemType, state string) {
	if s.metric == nil {
		return
	}

	s.metric.Add(ctx, 1, metric.WithAttributes(
		attribute.String("source", source),
		attribute.String("kind", kind),
		attribute.String("itemType", itemType),
		attribute.String("state", state),
	))
}
