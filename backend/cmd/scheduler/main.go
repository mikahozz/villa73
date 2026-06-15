package main

import (
	"context"
	"os"
	"time"

	"github.com/mikahozz/gohome/integrations/shelly"
	"github.com/mikahozz/gohome/integrations/sun"
	"github.com/rs/zerolog/log"
)

var zone, _ = time.LoadLocation("Europe/Helsinki")

var sunDataInstance *sun.SunData

func init() {
	var err error
	sunDataInstance, err = sun.NewSunData()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load sun data")
	}
}

func getSunriseTimeToday() time.Time {
	now := time.Now().In(zone) // Get current time in Helsinki timezone
	data := sunDataInstance.GetSunDataForSingleDate(now)
	return data.Sunrise
}

func getSunsetTimeToday() time.Time {
	now := time.Now().In(zone) // Get current time in Helsinki timezone
	data := sunDataInstance.GetSunDataForSingleDate(now)
	return data.Sunset
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("panic", r).
				Stack().
				Msg("Fatal panic occurred")
			os.Exit(1)
		}
	}()

	scheduler := NewScheduler()
	scheduler.AddSchedule(&DailySchedule{
		Name:     "Night lights ON at sunset",
		Category: "night_lights",
		Trigger: Trigger{
			Time: getSunsetTimeToday,
		},
		Action: func(ctx context.Context) error { return shelly.TurnOn(ctx) },
	})
	scheduler.AddSchedule(&DailySchedule{
		Name:     "Night lights OFF at 23:00",
		Category: "night_lights",
		Trigger: Trigger{
			Time: func() time.Time {
				now := time.Now().In(zone) // Get current time in Helsinki timezone
				return time.Date(now.Year(), now.Month(), now.Day(), 23, 0, 0, 0, zone)
			},
		},
		Action: func(ctx context.Context) error { return shelly.TurnOff(ctx) },
	})

	scheduler.AddSchedule(&DailySchedule{
		Name:     "Morning lights ON at 6:45",
		Category: "night_lights",
		Trigger: Trigger{
			Time: func() time.Time {
				now := time.Now().In(zone) // Get current time in Helsinki timezone
				return time.Date(now.Year(), now.Month(), now.Day(), 6, 45, 0, 0, zone)
			},
		},
		Action: func(ctx context.Context) error { return shelly.TurnOn(ctx) },
	})
	// Morning light schedules share a category so they execute in insertion order.
	// If both are due in the same evaluation cycle, sunrise OFF runs after 6:45 ON
	// and can intentionally override it.
	scheduler.AddSchedule(&DailySchedule{
		Name:     "Morning lights OFF at sunrise",
		Category: "night_lights",
		Trigger: Trigger{
			Time: getSunriseTimeToday,
		},
		Action: func(ctx context.Context) error { return shelly.TurnOff(ctx) },
	})

	// Cabin bookings refresh: run docker compose service to scrape and store bookings.
	scheduler.AddSchedule(&DailySchedule{
		Name: "Cabin bookings refresh at 06:30",
		Trigger: Trigger{
			Time: func() time.Time {
				now := time.Now().In(zone)
				return time.Date(now.Year(), now.Month(), now.Day(), 6, 30, 0, 0, zone)
			},
		},
		Action: func(ctx context.Context) error {
			return runShellCommand(ctx, "/homeapp73-docker", "docker", "compose", "run", "--rm", "cabinbookings-refresh")
		},
	})

	scheduler.Start()
	defer scheduler.Stop()

	// Keep the main function running
	select {}
}
