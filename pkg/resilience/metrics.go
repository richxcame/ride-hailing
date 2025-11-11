package resilience

import (
	"strconv"
	"sync/atomic"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/sony/gobreaker"
)

var (
	// Circuit Breaker Metrics
	breakerStateGauge = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "circuit_breaker_state",
		Help: "Current state of circuit breakers (0=closed, 0.5=half-open, 1=open)",
	}, []string{"breaker"})

	breakerRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuit_breaker_requests_total",
		Help: "Total number of operations executed through a circuit breaker",
	}, []string{"breaker"})

	breakerFailuresTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuit_breaker_failures_total",
		Help: "Total number of circuit breaker executions that resulted in an error",
	}, []string{"breaker"})

	breakerFallbacksTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuit_breaker_fallbacks_total",
		Help: "Total number of times breaker fallbacks were triggered because the breaker was open",
	}, []string{"breaker"})

	breakerStateTransitions = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "circuit_breaker_state_changes_total",
		Help: "Total number of circuit breaker state transitions",
	}, []string{"breaker", "from", "to"})

	// Retry Metrics
	retryAttemptsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "retry_attempts_total",
		Help: "Total number of retry attempts across all operations",
	}, []string{"operation", "result"})

	retryOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "retry_operation_duration_seconds",
		Help:    "Duration of retry operations including all attempts",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to ~4s
	}, []string{"operation", "result"})

	retryAttemptsHistogram = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "retry_attempts_count",
		Help:    "Number of attempts before success or final failure",
		Buckets: []float64{1, 2, 3, 4, 5, 10},
	}, []string{"operation", "result"})

	retryBackoffDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "retry_backoff_duration_seconds",
		Help:    "Duration of backoff delays during retries",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 10ms to ~5s
	}, []string{"operation"})

	breakerIDCounter uint64
)

func nextBreakerName(base string) string {
	if base != "" {
		return base
	}
	id := atomic.AddUint64(&breakerIDCounter, 1)
	return "breaker-" + strconv.FormatUint(id, 10)
}

func breakerStateValue(state gobreaker.State) float64 {
	switch state {
	case gobreaker.StateClosed:
		return 0
	case gobreaker.StateHalfOpen:
		return 0.5
	case gobreaker.StateOpen:
		return 1
	default:
		return -1
	}
}

func recordBreakerState(name string, state gobreaker.State) {
	breakerStateGauge.WithLabelValues(name).Set(breakerStateValue(state))
}

func recordBreakerStateChange(name string, from, to gobreaker.State) {
	breakerStateTransitions.WithLabelValues(name, from.String(), to.String()).Inc()
	recordBreakerState(name, to)
}

func recordBreakerRequest(name string) {
	breakerRequestsTotal.WithLabelValues(name).Inc()
}

func recordBreakerFailure(name string) {
	breakerFailuresTotal.WithLabelValues(name).Inc()
}

func recordBreakerFallback(name string) {
	breakerFallbacksTotal.WithLabelValues(name).Inc()
}

// Retry metrics recording functions

// RecordRetryAttempt records a retry attempt (success or failure)
func RecordRetryAttempt(operation string, success bool) {
	result := "failure"
	if success {
		result = "success"
	}
	retryAttemptsTotal.WithLabelValues(operation, result).Inc()
}

// RecordRetryOperation records the overall retry operation duration and attempt count
func RecordRetryOperation(operation string, durationSeconds float64, attempts int, success bool) {
	result := "failure"
	if success {
		result = "success"
	}

	retryOperationDuration.WithLabelValues(operation, result).Observe(durationSeconds)
	retryAttemptsHistogram.WithLabelValues(operation, result).Observe(float64(attempts))
}

// RecordRetryBackoff records a backoff delay duration
func RecordRetryBackoff(operation string, durationSeconds float64) {
	retryBackoffDuration.WithLabelValues(operation).Observe(durationSeconds)
}
