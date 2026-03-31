package reporter

import (
	"os"
	"strings"

	"github.com/rs/zerolog"
	"github.com/lodatang/apihealth/internal/checker"
)

// SetupLogger configures file logging
func SetupLogger(logFile string) (zerolog.Logger, error) {
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return zerolog.Logger{}, err
	}

	logger := zerolog.New(file).With().Timestamp().Logger()
	return logger, nil
}

// LogResult logs a check result with full details
func LogResult(logger zerolog.Logger, result checker.Result) {
	event := logger.Info().
		Str("target", result.Name).
		Str("model", result.Model).
		Bool("success", result.Success).
		Int("status_code", result.StatusCode).
		Dur("duration", result.Duration)

	if result.Error != nil {
		event = event.
			Str("error_type", result.Error.Type.String()).
			Str("error_message", result.Error.Message)

		if result.Error.Err != nil {
			event = event.Str("error_detail", result.Error.Err.Error())
		}
	}

	event.Msg("Health check completed")
}

// SanitizeAPIKey masks API key for logging (show only first 8 chars)
func SanitizeAPIKey(key string) string {
	if len(key) <= 8 {
		return strings.Repeat("*", len(key))
	}
	return key[:8] + strings.Repeat("*", len(key)-8)
}
