package main

import (
	"fmt"
	"net/http"
	"os"
	"service-b/handlers"

	"context"
	"log"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

func main() {
	// Configure OpenTelemetry
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/weather", otelhttp.NewHandler(http.HandlerFunc(handlers.GetWeather), "GetWeather").ServeHTTP)

	http.ListenAndServe(":8082", r)
}

func initTracer() (*sdktrace.TracerProvider, error) {
	endpoint := os.Getenv("ZIPKIN_ENDPOINT")
	exporter, err := zipkin.New(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create zipkin exporter: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String("service-b"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}
