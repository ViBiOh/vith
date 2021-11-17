package vith

func (a App) increaseMetric(source, kind, state string) {
	if a.metric == nil {
		return
	}

	a.metric.WithLabelValues(source, kind, state).Inc()
}
