package tracing

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Database span attributes
const (
	DBSystemKey      = attribute.Key("db.system")
	DBStatementKey   = attribute.Key("db.statement")
	DBOperationKey   = attribute.Key("db.operation")
	DBNameKey        = attribute.Key("db.name")
	DBTableKey       = attribute.Key("db.sql.table")
	DBRowsAffected   = attribute.Key("db.rows_affected")
)

// Redis span attributes
const (
	RedisCommandKey = attribute.Key("redis.command")
	RedisKeyKey     = attribute.Key("redis.key")
)

// HTTP span attributes
const (
	HTTPMethodKey     = attribute.Key("http.method")
	HTTPURLKey        = attribute.Key("http.url")
	HTTPStatusKey     = attribute.Key("http.status_code")
	HTTPRouteKey      = attribute.Key("http.route")
	HTTPClientIPKey   = attribute.Key("http.client_ip")
	HTTPUserAgentKey  = attribute.Key("http.user_agent")
	HTTPRequestIDKey  = attribute.Key("http.request_id")
)

// Business logic span attributes
const (
	UserIDKey       = attribute.Key("user.id")
	RideIDKey       = attribute.Key("ride.id")
	DriverIDKey     = attribute.Key("driver.id")
	PaymentIDKey    = attribute.Key("payment.id")
	FareAmountKey   = attribute.Key("fare.amount")
	DistanceKey     = attribute.Key("distance.meters")
	DurationKey     = attribute.Key("duration.seconds")
	LocationLatitudeKey  = attribute.Key("location.latitude")
	LocationLongitudeKey = attribute.Key("location.longitude")
)

// TraceDBQuery wraps a database query with tracing
func TraceDBQuery(ctx context.Context, tracerName, operation, query string, fn func() error) error {
	ctx, span := StartSpan(ctx, tracerName, fmt.Sprintf("db.%s", operation),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	span.SetAttributes(
		DBSystemKey.String("postgresql"),
		DBOperationKey.String(operation),
		DBStatementKey.String(query),
	)

	err := fn()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// TraceDBQueryWithResult wraps a database query with tracing and captures rows affected
func TraceDBQueryWithResult(ctx context.Context, tracerName, operation, query string, fn func() (sql.Result, error)) (sql.Result, error) {
	ctx, span := StartSpan(ctx, tracerName, fmt.Sprintf("db.%s", operation),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	span.SetAttributes(
		DBSystemKey.String("postgresql"),
		DBOperationKey.String(operation),
		DBStatementKey.String(query),
	)

	result, err := fn()
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	if result != nil {
		if rowsAffected, err := result.RowsAffected(); err == nil {
			span.SetAttributes(DBRowsAffected.Int64(rowsAffected))
		}
	}

	span.SetStatus(codes.Ok, "")
	return result, nil
}

// TraceRedisCommand wraps a Redis command with tracing
func TraceRedisCommand(ctx context.Context, tracerName, command, key string, fn func() error) error {
	ctx, span := StartSpan(ctx, tracerName, fmt.Sprintf("redis.%s", command),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	span.SetAttributes(
		attribute.String("db.system", "redis"),
		RedisCommandKey.String(command),
		RedisKeyKey.String(key),
	)

	err := fn()
	if err != nil && err != redis.Nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// TraceHTTPClient wraps an HTTP client call with tracing
func TraceHTTPClient(ctx context.Context, tracerName, method, url string, fn func() (int, error)) (int, error) {
	ctx, span := StartSpan(ctx, tracerName, fmt.Sprintf("HTTP %s", method),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	span.SetAttributes(
		HTTPMethodKey.String(method),
		HTTPURLKey.String(url),
	)

	statusCode, err := fn()

	span.SetAttributes(HTTPStatusKey.Int(statusCode))

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else if statusCode >= 400 {
		span.SetStatus(codes.Error, fmt.Sprintf("HTTP %d", statusCode))
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return statusCode, err
}

// TraceBusinessLogic wraps business logic with tracing
func TraceBusinessLogic(ctx context.Context, tracerName, operation string, attrs []attribute.KeyValue, fn func(context.Context) error) error {
	ctx, span := StartSpan(ctx, tracerName, operation,
		trace.WithSpanKind(trace.SpanKindInternal),
	)
	defer span.End()

	if len(attrs) > 0 {
		span.SetAttributes(attrs...)
	}

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	span.SetAttributes(
		attribute.Int64("duration_ms", duration.Milliseconds()),
	)

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// TraceExternalAPI wraps external API calls with tracing
func TraceExternalAPI(ctx context.Context, tracerName, serviceName, operation string, fn func(context.Context) error) error {
	ctx, span := StartSpan(ctx, tracerName, fmt.Sprintf("%s.%s", serviceName, operation),
		trace.WithSpanKind(trace.SpanKindClient),
	)
	defer span.End()

	span.SetAttributes(
		attribute.String("external.service", serviceName),
		attribute.String("external.operation", operation),
	)

	err := fn(ctx)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}

	return err
}

// Helper function to create ride-specific attributes
func RideAttributes(rideID, userID, driverID string) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 3)
	if rideID != "" {
		attrs = append(attrs, RideIDKey.String(rideID))
	}
	if userID != "" {
		attrs = append(attrs, UserIDKey.String(userID))
	}
	if driverID != "" {
		attrs = append(attrs, DriverIDKey.String(driverID))
	}
	return attrs
}

// Helper function to create payment-specific attributes
func PaymentAttributes(paymentID string, amount float64) []attribute.KeyValue {
	attrs := make([]attribute.KeyValue, 0, 2)
	if paymentID != "" {
		attrs = append(attrs, PaymentIDKey.String(paymentID))
	}
	if amount > 0 {
		attrs = append(attrs, FareAmountKey.Float64(amount))
	}
	return attrs
}

// Helper function to create location-specific attributes
func LocationAttributes(latitude, longitude float64) []attribute.KeyValue {
	return []attribute.KeyValue{
		LocationLatitudeKey.Float64(latitude),
		LocationLongitudeKey.Float64(longitude),
	}
}
