package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/zipkin"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
)

type Input struct {
	CEP string `json:"cep"`
}

type WeatherResponse struct {
	City  string  `json:"city"`
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

func main() {
	// Configure OpenTelemetry
	tp, err := initTracer()
	if err != nil {
		log.Fatalf("failed to initialize tracer: %v", err)
	}
	defer func() { _ = tp.Shutdown(context.Background()) }()

	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/cep", otelhttp.NewHandler(http.HandlerFunc(handleCEP), "handleCEP").ServeHTTP)

	log.Println("Starting server on :8081")
	http.ListenAndServe(":8081", r)
}

func handleCEP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid input", http.StatusUnprocessableEntity)
		return
	}

	if !isValidCEP(input.CEP) {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	serviceBURL := os.Getenv("SERVICE_B_URL")
	reqBody, err := json.Marshal(input)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	req, err := http.NewRequestWithContext(ctx, "POST", serviceBURL, bytes.NewBuffer(reqBody))
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Transport: otelhttp.NewTransport(http.DefaultTransport)}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode == http.StatusOK {
		var weatherResponse WeatherResponse
		if err := json.NewDecoder(resp.Body).Decode(&weatherResponse); err != nil {
			http.Error(w, "error decoding response from service B", http.StatusInternalServerError)
			return
		}
		if err := json.NewEncoder(w).Encode(weatherResponse); err != nil {
			http.Error(w, "error encoding response", http.StatusInternalServerError)
			return
		}
	} else {
		http.Error(w, http.StatusText(resp.StatusCode), resp.StatusCode)
	}
}

func isValidCEP(cep string) bool {
	re := regexp.MustCompile(`^\d{8}$`)
	return re.MatchString(cep)
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
			semconv.ServiceNameKey.String("service-a"),
		)),
	)
	otel.SetTracerProvider(tp)
	return tp, nil
}
