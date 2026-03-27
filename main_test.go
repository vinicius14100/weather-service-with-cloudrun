package main

import (
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
)

func assertFloatEqual(t *testing.T, got, want float64) {
	t.Helper()
	const eps = 1e-9
	if math.Abs(got-want) > eps {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestIsValidCEP(t *testing.T) {
	tests := []struct {
		name string
		cep  string
		want bool
	}{
		{"Valid CEP", "01153000", true},
		{"Valid CEP with zeros", "00000000", true},
		{"Invalid CEP - too short", "123456", false},
		{"Invalid CEP - too long", "123456789", false},
		{"Invalid CEP - with letters", "1234567a", false},
		{"Invalid CEP - with special chars", "12345-678", false},
		{"Empty CEP", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isValidCEP(tt.cep); got != tt.want {
				t.Errorf("isValidCEP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCelsiusToFahrenheit(t *testing.T) {
	tests := []struct {
		name     string
		celsius  float64
		expected float64
	}{
		{"0°C", 0, 32},
		{"100°C", 100, 212},
		{"-40°C", -40, -40},
		{"25°C", 25, 77},
		{"28.5°C", 28.5, 83.3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := celsiusToFahrenheit(tt.celsius)
			assertFloatEqual(t, result, tt.expected)
		})
	}
}

func TestCelsiusToKelvin(t *testing.T) {
	tests := []struct {
		name     string
		celsius  float64
		expected float64
	}{
		{"0°C", 0, 273.15},
		{"100°C", 100, 373.15},
		{"-273.15°C", -273.15, 0},
		{"25°C", 25, 298.15},
		{"28.5°C", 28.5, 301.65},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := celsiusToKelvin(tt.celsius)
			assertFloatEqual(t, result, tt.expected)
		})
	}
}

func TestWeatherHandler(t *testing.T) {
	tests := []struct {
		name           string
		cep            string
		expectedStatus int
		expectedBody   string
	}{
		{"Invalid CEP - empty", "", 422, "invalid zipcode"},
		{"Invalid CEP - too short", "123456", 422, "invalid zipcode"},
		{"Invalid CEP - with letters", "1234567a", 422, "invalid zipcode"},
	}

	svc := &Service{HTTPClient: http.DefaultClient}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/weather?cep="+tt.cep, nil)
			w := httptest.NewRecorder()

			svc.WeatherHandler(w, req)

			resp := w.Result()
			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.StatusCode)
			}

			if tt.expectedBody != "" {
				body := w.Body.String()
				if body != tt.expectedBody {
					t.Errorf("Expected body %q, got %q", tt.expectedBody, body)
				}
			}
		})
	}
}

func TestWeatherHandler_Success200(t *testing.T) {
	viaCEPServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/ws/01153000/json/" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"localidade":"Sao Paulo"}`))
	}))
	defer viaCEPServer.Close()

	weatherServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/current.json" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.URL.Query().Get("key") != "test-key" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if r.URL.Query().Get("q") != "Sao Paulo" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"current":{"temp_c":28.5}}`))
	}))
	defer weatherServer.Close()

	svc := &Service{
		HTTPClient:     http.DefaultClient,
		ViaCEPBaseURL:  viaCEPServer.URL,
		WeatherBaseURL: weatherServer.URL,
		WeatherAPIKey:  "test-key",
	}

	req := httptest.NewRequest("GET", "/weather?cep=01153000", nil)
	w := httptest.NewRecorder()
	svc.WeatherHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.StatusCode)
	}

	var got WeatherResponse
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	assertFloatEqual(t, got.TempC, 28.5)
	assertFloatEqual(t, got.TempF, 83.3)
	assertFloatEqual(t, got.TempK, 301.65)
}

func TestWeatherHandler_ZipcodeNotFound404(t *testing.T) {
	viaCEPServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"erro":true}`))
	}))
	defer viaCEPServer.Close()

	weatherServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"current":{"temp_c":28.5}}`))
	}))
	defer weatherServer.Close()

	svc := &Service{
		HTTPClient:     http.DefaultClient,
		ViaCEPBaseURL:  viaCEPServer.URL,
		WeatherBaseURL: weatherServer.URL,
		WeatherAPIKey:  "test-key",
	}

	req := httptest.NewRequest("GET", "/weather?cep=99999999", nil)
	w := httptest.NewRecorder()
	svc.WeatherHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
	if body := w.Body.String(); body != "can not find zipcode" {
		t.Fatalf("expected body %q, got %q", "can not find zipcode", body)
	}
}

func TestWeatherHandler_ZipcodeNotFound404_ViaCEPStringTrue(t *testing.T) {
	viaCEPServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"erro":"true"}`))
	}))
	defer viaCEPServer.Close()

	weatherServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"current":{"temp_c":28.5}}`))
	}))
	defer weatherServer.Close()

	svc := &Service{
		HTTPClient:     http.DefaultClient,
		ViaCEPBaseURL:  viaCEPServer.URL,
		WeatherBaseURL: weatherServer.URL,
		WeatherAPIKey:  "test-key",
	}

	req := httptest.NewRequest("GET", "/weather?cep=99999999", nil)
	w := httptest.NewRecorder()
	svc.WeatherHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
	if body := w.Body.String(); body != "can not find zipcode" {
		t.Fatalf("expected body %q, got %q", "can not find zipcode", body)
	}
}

func TestWeatherHandler_ViaCEP404NonJSON_ReturnsZipcodeNotFound404(t *testing.T) {
	viaCEPServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Not Found"))
	}))
	defer viaCEPServer.Close()

	weatherServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"current":{"temp_c":28.5}}`))
	}))
	defer weatherServer.Close()

	svc := &Service{
		HTTPClient:     http.DefaultClient,
		ViaCEPBaseURL:  viaCEPServer.URL,
		WeatherBaseURL: weatherServer.URL,
		WeatherAPIKey:  "test-key",
	}

	req := httptest.NewRequest("GET", "/weather?cep=99999999", nil)
	w := httptest.NewRecorder()
	svc.WeatherHandler(w, req)

	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.StatusCode)
	}
	if body := w.Body.String(); body != "can not find zipcode" {
		t.Fatalf("expected body %q, got %q", "can not find zipcode", body)
	}
}
