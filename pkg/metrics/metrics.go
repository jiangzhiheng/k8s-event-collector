package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	SearchK8sEventServerTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: "k8s_event",
			Name: "search_event_server_total",
			Help: "call grpc interface total",
		},[]string{"eventNamespace"})
)

func AddSearchK8sEventServerTotal(eventNamespace string){
	SearchK8sEventServerTotal.WithLabelValues(eventNamespace).Inc()
}

