package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

// Open-Meteo API — free, no auth required.
// Fetches current weather for Lisbon, Porto, Bern and Geneva in a single call
// using the multi-location endpoint.
const openMeteoURL = "https://api.open-meteo.com/v1/forecast?" +
	"latitude=38.7167,41.1496,46.9481,46.2044" +
	"&longitude=-9.1333,-8.6110,7.4474,6.1432" +
	"&current=temperature_2m,relative_humidity_2m,wind_speed_10m,weather_code" +
	"&timezone=auto"

// City names mapped by index in the API response.
var cities = []string{"lisbon", "porto", "bern", "geneva"}

// openMeteoResponse represents the top-level JSON array returned when
// querying multiple locations. Open-Meteo returns a JSON array when
// multiple latitudes/longitudes are provided.
type openMeteoResponse struct {
	Current struct {
		Temperature  float64 `json:"temperature_2m"`
		Humidity     float64 `json:"relative_humidity_2m"`
		WindSpeed    float64 `json:"wind_speed_10m"`
		WeatherCode  int     `json:"weather_code"`
	} `json:"current"`
}

type cityWeather struct {
	Temperature float64
	Humidity    float64
	WindSpeed   float64
	WeatherCode int
}

type metrics struct {
	mu             sync.RWMutex
	weather        map[string]cityWeather // keyed by city name
	scrapeCount    float64
	scrapeErrors   float64
	lastScrapeTime float64 // unix timestamp
}

var m = &metrics{
	weather: make(map[string]cityWeather),
}

func scrape() {
	resp, err := http.Get(openMeteoURL)
	if err != nil {
		m.mu.Lock()
		m.scrapeErrors++
		m.mu.Unlock()
		log.Printf("error fetching weather: %v", err)
		return
	}
	defer resp.Body.Close()

	// Open-Meteo returns a JSON array when multiple locations are queried.
	var results []openMeteoResponse
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		m.mu.Lock()
		m.scrapeErrors++
		m.mu.Unlock()
		log.Printf("error decoding weather response: %v", err)
		return
	}

	if len(results) != len(cities) {
		m.mu.Lock()
		m.scrapeErrors++
		m.mu.Unlock()
		log.Printf("unexpected number of results: got %d, want %d", len(results), len(cities))
		return
	}

	m.mu.Lock()
	for i, city := range cities {
		c := results[i].Current
		m.weather[city] = cityWeather{
			Temperature: c.Temperature,
			Humidity:    c.Humidity,
			WindSpeed:   c.WindSpeed,
			WeatherCode: c.WeatherCode,
		}
		log.Printf("scraped %s: temp=%.1f°C humidity=%.0f%% wind=%.1fkm/h code=%d",
			city, c.Temperature, c.Humidity, c.WindSpeed, c.WeatherCode)
	}
	m.scrapeCount++
	m.lastScrapeTime = float64(time.Now().Unix())
	m.mu.Unlock()
}

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	w.Header().Set("Content-Type", "text/plain; version=0.0.4")

	// Per-city gauges
	for _, city := range cities {
		cw, ok := m.weather[city]
		if !ok {
			continue
		}

		fmt.Fprintf(w, "# HELP weather_temperature_celsius Current temperature in degrees Celsius\n")
		fmt.Fprintf(w, "# TYPE weather_temperature_celsius gauge\n")
		fmt.Fprintf(w, "weather_temperature_celsius{city=%q} %g\n\n", city, cw.Temperature)

		fmt.Fprintf(w, "# HELP weather_relative_humidity_percent Current relative humidity percentage\n")
		fmt.Fprintf(w, "# TYPE weather_relative_humidity_percent gauge\n")
		fmt.Fprintf(w, "weather_relative_humidity_percent{city=%q} %g\n\n", city, cw.Humidity)

		fmt.Fprintf(w, "# HELP weather_wind_speed_kmh Current wind speed in km/h\n")
		fmt.Fprintf(w, "# TYPE weather_wind_speed_kmh gauge\n")
		fmt.Fprintf(w, "weather_wind_speed_kmh{city=%q} %g\n\n", city, cw.WindSpeed)

		fmt.Fprintf(w, "# HELP weather_code WMO weather interpretation code\n")
		fmt.Fprintf(w, "# TYPE weather_code gauge\n")
		fmt.Fprintf(w, "weather_code{city=%q} %d\n\n", city, cw.WeatherCode)
	}

	// Scraper operational metrics
	fmt.Fprintf(w, "# HELP weather_scrape_total Total number of successful scrapes\n")
	fmt.Fprintf(w, "# TYPE weather_scrape_total counter\n")
	fmt.Fprintf(w, "weather_scrape_total %g\n\n", m.scrapeCount)

	fmt.Fprintf(w, "# HELP weather_scrape_errors_total Total number of failed scrapes\n")
	fmt.Fprintf(w, "# TYPE weather_scrape_errors_total counter\n")
	fmt.Fprintf(w, "weather_scrape_errors_total %g\n\n", m.scrapeErrors)

	fmt.Fprintf(w, "# HELP weather_last_scrape_timestamp_seconds Unix timestamp of the last successful scrape\n")
	fmt.Fprintf(w, "# TYPE weather_last_scrape_timestamp_seconds gauge\n")
	fmt.Fprintf(w, "weather_last_scrape_timestamp_seconds %g\n", m.lastScrapeTime)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "ok")
}

func main() {
	// Initial scrape immediately, then every 5 minutes.
	// Weather data doesn't change as frequently as HN — 5min is plenty.
	scrape()
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			scrape()
		}
	}()

	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/healthz", healthHandler)

	log.Println("starting weather scraper on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("server error: %v", err)
	}
}