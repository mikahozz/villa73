package mock

import (
	"fmt"
	"time"
)

func IndoorDevUpstairs() (string, error) {
	return fmt.Sprintf(`{"battery":100.0,"humidity":27.4,"temperature":22.5,"time":"%s"}`, time.Now().UTC().Format(time.RFC3339Nano)), nil
}
