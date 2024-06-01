package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
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

type WeatherAPIResponse struct {
	Location struct {
		Name string `json:"name"`
	} `json:"location"`
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

func GetWeather(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tracer := otel.Tracer("service-b")

	_, span := tracer.Start(ctx, "GetWeather")
	defer span.End()

	var input Input
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid input", http.StatusUnprocessableEntity)
		return
	}

	if !isValidCEP(input.CEP) {
		http.Error(w, "invalid zipcode", http.StatusUnprocessableEntity)
		return
	}

	location, err := getLocation(ctx, input.CEP)
	if err != nil {
		http.Error(w, "can not find zipcode", http.StatusNotFound)
		return
	}

	tempC, err := getTemperature(ctx, location)
	if err != nil {
		http.Error(w, "error fetching temperature", http.StatusInternalServerError)
		return
	}

	tempF := tempC*1.8 + 32
	tempK := tempC + 273.15

	response := WeatherResponse{
		City:  location,
		TempC: tempC,
		TempF: tempF,
		TempK: tempK,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func isValidCEP(cep string) bool {
	re := regexp.MustCompile(`^\d{8}$`)
	return re.MatchString(cep)
}

func getLocation(ctx context.Context, cep string) (string, error) {
	tracer := otel.Tracer("service-b")

	ctx, span := tracer.Start(ctx, "getLocation")
	span.SetAttributes(attribute.String("cep", cep))
	defer span.End()

	url := fmt.Sprintf("https://viacep.com.br/ws/%s/json/", cep)
	log.Printf("Fetching location from URL: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected response status: %d", resp.StatusCode)
		return "", fmt.Errorf("error fetching location")
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode JSON response: %v", err)
		return "", err
	}

	if _, ok := result["erro"]; ok {
		log.Printf("CEP not found: %s", cep)
		return "", fmt.Errorf("CEP not found")
	}

	location, ok := result["localidade"].(string)
	if !ok {
		log.Printf("Failed to parse location from response: %+v", result)
		return "", fmt.Errorf("error parsing location")
	}

	span.SetAttributes(attribute.String("location", location))
	return location, nil
}

func getTemperature(ctx context.Context, location string) (float64, error) {
	tracer := otel.Tracer("service-b")

	ctx, span := tracer.Start(ctx, "getTemperature")
	span.SetAttributes(attribute.String("location", location))
	defer span.End()

	apiKey := "98a42a15c266432a98b25526240106"
	encodedLocation := url.QueryEscape(location)
	url := fmt.Sprintf("http://api.weatherapi.com/v1/current.json?key=%s&q=%s", apiKey, encodedLocation)
	log.Printf("Fetching temperature from URL: %s", url)
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("HTTP request failed: %v", err)
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Unexpected response status: %d", resp.StatusCode)
		return 0, fmt.Errorf("error fetching temperature")
	}

	var result WeatherAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		log.Printf("Failed to decode JSON response: %v", err)
		return 0, err
	}

	span.SetAttributes(attribute.Float64("tempC", result.Current.TempC))
	return result.Current.TempC, nil
}
