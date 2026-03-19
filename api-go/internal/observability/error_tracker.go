package observability

import (
	"github.com/getsentry/sentry-go"

	"github.com/getlago/lago/api-go/internal/result"
)

// CaptureErrorResult sends the error from res to Sentry with optional extra context.
func CaptureErrorResult(res result.AnyResult) {
	CaptureErrorResultWithExtra(res, "", nil)
}

// CaptureErrorResultWithExtra sends the error from res to Sentry with extra key/value.
func CaptureErrorResultWithExtra(res result.AnyResult, extraKey string, extraValue any) {
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetExtra("error_code", res.ErrorCode())
		scope.SetExtra("error_message", res.ErrorMessage())
		if extraKey != "" {
			scope.SetExtra(extraKey, extraValue)
		}
		sentry.CaptureException(res.Error())
	})
}

// CaptureError sends err to Sentry.
func CaptureError(err error) {
	sentry.CaptureException(err)
}
