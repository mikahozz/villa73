package spot

// NoDataError represents an error when no data is available
type NoDataError struct {
	Message string
}

// Error implements the error interface for NoDataError
func (e NoDataError) Error() string {
	return e.Message
}
