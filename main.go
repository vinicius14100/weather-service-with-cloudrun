package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
)

var errZipcodeNotFound = errors.New("zipcode not found")

type WeatherResponse struct {
	TempC float64 `json:"temp_C"`
	TempF float64 `json:"temp_F"`
	TempK float64 `json:"temp_K"`
}

type ViaCEPResponse struct {
	Localidade string `json:"localidade"`
	Erro       bool   `json:"erro"`
}

type WeatherAPIResponse struct {
	Current struct {
		TempC float64 `json:"temp_c"`
	} `json:"current"`
}

type Service struct {
	HTTPClient     *http.Client
	ViaCEPBaseURL  string
	WeatherBaseURL string
	WeatherAPIKey  string
}

func main() {
	svc := NewServiceFromEnv()
	http.HandleFunc("/weather", svc.WeatherHandler)
	log.Println("Server starting on port 8080...")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func NewServiceFromEnv() *Service {
	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		apiKey = "YOUR_WEATHER_API_KEY" // Fallback for local development
	}

	return &Service{
		HTTPClient:     http.DefaultClient,
		ViaCEPBaseURL:  "https://viacep.com.br",
		WeatherBaseURL: "http://api.weatherapi.com",
		WeatherAPIKey:  apiKey,
	}
}

func (s *Service) WeatherHandler(w http.ResponseWriter, r *http.Request) {
	cep := r.URL.Query().Get("cep")
	if cep == "" {
		writePlainError(w, http.StatusUnprocessableEntity, "invalid zipcode")
		return
	}

	if !isValidCEP(cep) {
		writePlainError(w, http.StatusUnprocessableEntity, "invalid zipcode")
		return
	}

	city, err := s.getLocationByCEP(cep)
	if err != nil {
		if errors.Is(err, errZipcodeNotFound) {
			writePlainError(w, http.StatusNotFound, "can not find zipcode")
			return
		}
		writePlainError(w, http.StatusInternalServerError, "internal error")
		return
	}

	tempC, err := s.getTemperatureByCity(city)
	if err != nil {
		writePlainError(w, http.StatusInternalServerError, "error getting weather data")
		return
	}

	response := WeatherResponse{
		TempC: tempC,
		TempF: celsiusToFahrenheit(tempC),
		TempK: celsiusToKelvin(tempC),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func writePlainError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(status)
	_, _ = w.Write([]byte(msg))
}

func isValidCEP(cep string) bool {
	if len(cep) != 8 {
		return false
	}
	match, _ := regexp.MatchString(`^\d{8}$`, cep)
	return match
}

func (s *Service) getLocationByCEP(cep string) (string, error) {
	endpoint := fmt.Sprintf("%s/ws/%s/json/", s.ViaCEPBaseURL, cep)
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return "", err
	}

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusBadRequest || resp.StatusCode == http.StatusNotFound {
		return "", errZipcodeNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("viacep returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var viaCEPResp ViaCEPResponse
	if err := json.Unmarshal(body, &viaCEPResp); err != nil {
		return "", err
	}

	if viaCEPResp.Erro || viaCEPResp.Localidade == "" {
		return "", errZipcodeNotFound
	}

	return viaCEPResp.Localidade, nil
}

func (s *Service) getTemperatureByCity(city string) (float64, error) {
	if s.WeatherAPIKey == "" {
		return 0, fmt.Errorf("missing weather api key")
	}

	u, err := url.Parse(s.WeatherBaseURL)
	if err != nil {
		return 0, err
	}
	u.Path = "/v1/current.json"
	q := u.Query()
	q.Set("key", s.WeatherAPIKey)
	q.Set("q", city)
	q.Set("aqi", "no")
	u.RawQuery = q.Encode()

	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return 0, err
	}

	resp, err := s.HTTPClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}

	var weatherResp WeatherAPIResponse
	if err := json.Unmarshal(body, &weatherResp); err != nil {
		return 0, err
	}

	return weatherResp.Current.TempC, nil
}

func celsiusToFahrenheit(c float64) float64 {
	return c*1.8 + 32
}

func celsiusToKelvin(c float64) float64 {
	return c + 273.15
}
