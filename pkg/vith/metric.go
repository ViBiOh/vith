package vith

func (a App) increaseMetric(source, kind, itemType, state string) {
	if a.metric == nil {
		return
	}

	a.metric.WithLabelValues(source, kind, itemType, state).Inc()
}
