package account

// ErrorType classifies upstream API errors for failover decisions.
type ErrorType int

const (
	// ErrorFatal means the error is permanent for this request; do not retry with another account.
	// Examples: 400 validation error, 403 forbidden, 413 payload too large, 422 unprocessable.
	ErrorFatal ErrorType = iota

	// ErrorRecoverable means the error is transient or account-specific; try the next account.
	// Examples: 429 rate limit, 402 quota exceeded, 500/502/503/504 upstream errors.
	ErrorRecoverable
)

// ClassifyError determines whether an upstream HTTP error is fatal or recoverable.
func ClassifyError(statusCode int) ErrorType {
	switch statusCode {
	case 400, 403, 413, 422:
		return ErrorFatal
	case 429, 402, 500, 502, 503, 504:
		return ErrorRecoverable
	default:
		if statusCode >= 400 && statusCode < 500 {
			return ErrorFatal // Client errors are generally not retryable
		}
		return ErrorRecoverable // Server errors might be transient
	}
}
