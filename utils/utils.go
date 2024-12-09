package utils

import (
	"errors"
	"math"
	"time"
)

// ExponentialBackoff retries a given operation with exponential backoff.
// Parameters:
//   - operation: A function that performs the operation to retry (e.g., client connection or JSON request).
//     The operation should return an error if it fails and nil on success.
//   - maxRetries: The maximum number of retry attempts.
//   - baseDelay: The base delay in milliseconds between retries.
//   - maxDelay: The maximum delay in milliseconds between retries.
//
// Returns an error if all retries fail.
func ExponentialBackoff(operation func() error, maxRetries int, baseDelay, maxDelay time.Duration) error {
	if maxRetries < 1 {
		return errors.New("maxRetries must be at least 1")
	}

	var err error
	for i := 0; i < maxRetries; i++ {
		err = operation()
		if err == nil {
			return nil
		}

		// Calculate delay with exponential backoff
		delay := baseDelay * time.Duration(math.Pow(2, float64(i)))
		if delay > maxDelay {
			delay = maxDelay
		}

		time.Sleep(delay)
	}
	return err
}
