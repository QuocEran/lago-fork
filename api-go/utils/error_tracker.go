package utils

import (
	"log/slog"

	"github.com/getsentry/sentry-go"
)

func CaptureErrorResult(res AnyResult) {
	CaptureErrorResultWithExtra(res, "", nil)
}

func CaptureErrorResultWithExtra(res AnyResult, extraKey string, extraValue any) {
	sentry.WithScope(func(scope *sentry.Scope) {
		scope.SetExtra("error_code", res.ErrorCode())
		scope.SetExtra("error_message", res.ErrorMessage())
		if extraKey != "" {
			scope.SetExtra(extraKey, extraValue)
		}
		sentry.CaptureException(res.Error())
	})
}

func CaptureError(err error) {
	sentry.CaptureException(err)
}

func LogAndPanic(err error, message string) {
	slog.Error(message, slog.String("error", err.Error()))
	CaptureError(err)
	panic(err.Error())
}
