package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"time"

	"github.com/mikahozz/gohome/config"
	"github.com/mikahozz/gohome/integrations/cal"
	"github.com/mikahozz/gohome/integrations/fmi"
	"github.com/mikahozz/gohome/integrations/spot"
	"github.com/mikahozz/gohome/integrations/sun"
	"github.com/mikahozz/gohome/mock"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

const port = ":6001"

// Handler functions for real data
type handlers struct {
	weatherNow     http.HandlerFunc
	weatherFore    http.HandlerFunc
	indoorTemp     http.HandlerFunc
	spotPrices     http.HandlerFunc
	calendarEvents http.HandlerFunc
	sunData        http.HandlerFunc
}

// Create real data handlers
func createRealHandlers() handlers {
	return handlers{
		weatherNow:     getWeatherData("101004", fmi.Observations),
		weatherFore:    getWeatherData("Tapanila,Helsinki", fmi.Forecast),
		indoorTemp:     jsonResponse(mock.IndoorDevUpstairs),
		spotPrices:     getSpotPrices(),
		calendarEvents: getCalendarEvents(),
		sunData:        getSunData(),
	}
}

// Create mock data handlers
func createMockHandlers() handlers {
	return handlers{
		weatherNow:     jsonResponse(mock.OutdoorWeathernNow),
		weatherFore:    jsonResponse(mock.OutdoorWeatherFore),
		indoorTemp:     jsonResponse(mock.IndoorDevUpstairs),
		spotPrices:     jsonResponse(mock.ElectricityPrices),
		calendarEvents: jsonResponse(mock.Events),
		sunData:        getSunData(), // We use hard code Helsinki data for now
	}
}

func jsonResponse(f func() (string, error)) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json, err := f()
		if err != nil {
			log.Error().Err(err).Msg("")
			http.Error(w, "Error occurred when performing request", http.StatusInternalServerError)
			return
		}
		fmt.Fprint(w, json)
	}
}

func getWeatherData(place string, requestType fmi.RequestType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		weather, err := fmi.GetWeatherData(fmi.StationId(place), requestType)
		if err != nil {
			log.Err(err).Msg("")
			http.Error(w, fmt.Sprintf("Error occurred in fetching weather data for %s", place), http.StatusInternalServerError)
			return
		}
		json, err := json.Marshal(weather.WeatherData)
		if err != nil {
			log.Err(err).Msg("")
			http.Error(w, fmt.Sprintf("Error occurred in fetching weather data for %s", place), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}
}

func getCalendarEvents() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from := cal.DateOffset{}
		to := cal.DateOffset{Days: 7}
		events, err := cal.GetFamilyCalendarEvents(from, to)
		if err != nil {
			log.Err(err).Msg("")
			http.Error(w, "Error occurred fetching calendar events", http.StatusInternalServerError)
			return
		}
		json, err := json.Marshal(events)
		if err != nil {
			log.Err(err).Msg("")
			http.Error(w, "Error occurred in json conversion of calendar events", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}
}

func getSpotPrices() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startStr := r.URL.Query().Get("start")
		endStr := r.URL.Query().Get("end")
		timeFormat := r.URL.Query().Get("timeFormat")

		// Default to UTC if timeFormat is not specified
		if timeFormat == "" {
			timeFormat = "utc"
		}

		// Get the location based on timeFormat
		var location *time.Location
		var err error
		switch timeFormat {
		case "utc":
			location = time.UTC
		case "local":
			location = time.Local
		default:
			location, err = time.LoadLocation(timeFormat)
			if err != nil {
				log.Error().Err(err).Msg("Invalid timezone format")
				http.Error(w, "Invalid timezone format", http.StatusBadRequest)
				return
			}
		}

		start, err := time.Parse(time.RFC3339, startStr)
		if err != nil {
			log.Err(err).Msg("")
			http.Error(w, "Invalid start time format. Use RFC3339.", http.StatusBadRequest)
			return
		}

		end, err := time.Parse(time.RFC3339, endStr)
		if err != nil {
			log.Err(err).Msg("")
			http.Error(w, "Invalid end time format. Use RFC3339.", http.StatusBadRequest)
			return
		}

		log.Info().Msgf("Getting spot prices for %s to %s in %s format", start, end, timeFormat)
		prices, err := spot.GetPrices(start, end, location)
		if err != nil {
			log.Error().Err(err).Msg("Error getting spot prices")
			http.Error(w, "Error occurred fetching spot prices", http.StatusInternalServerError)
			return
		}

		json, err := json.Marshal(prices.Prices)
		if err != nil {
			log.Error().Err(err).Msg("Error marshalling spot prices to JSON")
			http.Error(w, "Error occurred in JSON conversion of spot prices", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}
}

func getSunData() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse date parameters - only using YYYY-MM-DD format
		startStr := r.URL.Query().Get("start")
		endStr := r.URL.Query().Get("end")

		// Parse start date (required)
		start, err := time.Parse("2006-01-02", startStr)
		if err != nil {
			log.Err(err).Msg("Invalid start date format")
			http.Error(w, "Invalid start date format. Use YYYY-MM-DD (e.g., 2025-03-08).", http.StatusBadRequest)
			return
		}

		// Parse end date (optional)
		var end time.Time
		if endStr != "" {
			end, err = time.Parse("2006-01-02", endStr)
			if err != nil {
				log.Err(err).Msg("Invalid end date format")
				http.Error(w, "Invalid end date format. Use YYYY-MM-DD (e.g., 2025-03-10).", http.StatusBadRequest)
				return
			}
		}

		// Load sun data
		sunData, err := sun.NewSunData()

		if err != nil {
			log.Error().Err(err).Msg("Error loading sun data")
			http.Error(w, "Error occurred in loading sun data", http.StatusInternalServerError)
			return
		}

		// Get data for the single date (ignoring year, only returns one day)
		dailySunData := sunData.GetSunDataForDateRange(start, end)
		json, err := json.Marshal(dailySunData)
		if err != nil {
			log.Error().Err(err).Msg("Error marshalling sun data to JSON")
			http.Error(w, "Error occurred in JSON conversion of sun data", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(json)
	}
}

func printEndpoints() {
	fmt.Println("\nAvailable endpoints:")
	fmt.Println("-------------------")
	fmt.Printf("GET /weathernow                  - Current weather observations\n")
	fmt.Printf("    curl http://localhost:6001/api/weathernow\n")

	fmt.Printf("GET /weatherfore                 - Weather forecast\n")
	fmt.Printf("    curl http://localhost:6001/api/weatherfore\n")

	fmt.Printf("GET /indoor/dev_upstairs         - Indoor temperature\n")
	fmt.Printf("    curl http://localhost:6001/api/indoor/dev_upstairs\n")

	fmt.Printf("GET /electricity/prices          - Spot prices for time range (params: start, end, timeFormat)\n")
	fmt.Printf("    curl \"http://localhost:6001/api/electricity/prices?start=2024-03-20T00:00:00Z&end=2024-03-21T00:00:00Z&timeFormat=Europe/Helsinki\"\n")

	fmt.Printf("GET /api/events                  - Calendar events for next 7 days\n")
	fmt.Printf("    curl http://localhost:6001/api/events\n")

	fmt.Printf("GET /api/sun                    - Sunset and runrise info for date range (params: start, end)\n")
	fmt.Printf("    curl \"http://localhost:6001/api/sun?start=2025-03-20&end=2025-03-21\"\n")

	fmt.Printf("\nServer running on port %s\n\n", port)
}

func main() {
	// Add command line flag for mock mode
	useMock := flag.Bool("mock", false, "Use mock data instead of real integrations")
	flag.Parse()

	zerolog.TimeFieldFormat = time.RFC3339
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	config.LoadEnv()

	if *useMock {
		log.Info().Msg("Starting server in mock mode")
	} else {
		log.Info().Msg("Starting server with real integrations")
	}

	// Choose handlers based on mock flag
	h := createRealHandlers()
	if *useMock {
		h = createMockHandlers()
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/api/weathernow", h.weatherNow)
	mux.HandleFunc("/api/indoor/dev_upstairs", h.indoorTemp)
	mux.HandleFunc("/api/weatherfore", h.weatherFore)
	mux.HandleFunc("/api/electricity/prices", h.spotPrices)
	mux.HandleFunc("/api/events", h.calendarEvents)
	mux.HandleFunc("/api/sun", h.sunData)

	// Start server in a goroutine
	server := &http.Server{
		Addr:    port,
		Handler: loggingMiddleware(mux),
	}

	go func() {
		if err := server.ListenAndServe(); err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait a moment to ensure server is up
	time.Sleep(100 * time.Millisecond)

	// Print endpoints if server started successfully
	printEndpoints()

	// Keep the main goroutine running
	select {}
}
