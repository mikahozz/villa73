package cal

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	username     string
	password     string
	calUrl       string
	calName      string
	baseTimezone *time.Location
}

var config Config

func init() {
	err := godotenv.Load()
	if err != nil {
		panic(fmt.Sprintf("error loading .env file: %v", err))
	}
	config.username = os.ExpandEnv("$CAL_USERNAME")
	if config.username == "" {
		panic("CAL_USERNAME env not set")
	}
	config.password = os.ExpandEnv("$CAL_PASSWORD")
	if config.password == "" {
		panic("CAL_PASSWORD env not set")
	}
	config.calUrl = os.ExpandEnv("$CAL_URL")
	if config.calUrl == "" {
		panic("CAL_URL env not set")
	}
	config.calName = os.ExpandEnv("$CAL_NAME")
	if config.calName == "" {
		panic("CAL_NAME env not set")
	}
	zone := os.ExpandEnv("$CAL_BASE_TIMEZONE")
	if zone == "" {
		panic("CAL_BASE_TIMEZONE env not set")
	}
	config.baseTimezone, err = time.LoadLocation(zone)
	if err != nil {
		panic(fmt.Sprintf("error loading timezone. It should be a valid IANA Time Zone: %v", err))
	}
}
