package shared

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

// CustomErrorRule maps a matching error to a specific HTTP response.
type CustomErrorRule struct {
	Match   func(error) bool
	Status  int
	Code    string
	Details func(error) any // optional
}

// ServiceErrorClassifier defines how to map service errors to HTTP responses.
// Each handler package defines one and passes it to HandleServiceError.
// CustomErrors are checked first; if any Match(err), that response is sent.
// Optional IsNotFoundError / IsTransitionError predicates override slices when set.
type ServiceErrorClassifier struct {
	NotFoundErrors    []error
	ConflictErrors    []error
	TransitionErrors   []error
	CustomErrors      []CustomErrorRule
	IsNotFoundError   func(error) bool
	IsTransitionError func(error) bool
	IsValidationErr   func(error) bool
	NotFoundCode      string
	ConflictCode      string
	ConflictDetails   func(error) map[string]any
	TransitionCode    string
}

// HandleServiceError maps err to the appropriate HTTP status and standard error envelope using clf.
// Order: not-found -> 404, conflict -> 422 value_already_exist, transition -> 422 transition_error, validation -> 422 validation_error, default -> 500 internal_error.
func HandleServiceError(c *gin.Context, err error, clf ServiceErrorClassifier) {
	for _, rule := range clf.CustomErrors {
		if rule.Match != nil && rule.Match(err) {
			details := rule.Details
			if details == nil {
				RespondError(c, rule.Status, rule.Code, gin.H{})
			} else {
				RespondError(c, rule.Status, rule.Code, details(err))
			}
			return
		}
	}
	if clf.IsNotFoundError != nil && clf.IsNotFoundError(err) {
		code := clf.NotFoundCode
		if code == "" {
			code = "not_found"
		}
		RespondError(c, http.StatusNotFound, code, gin.H{})
		return
	}
	for _, e := range clf.NotFoundErrors {
		if errors.Is(err, e) {
			code := clf.NotFoundCode
			if code == "" {
				code = "not_found"
			}
			RespondError(c, http.StatusNotFound, code, gin.H{})
			return
		}
	}
	for _, e := range clf.ConflictErrors {
		if errors.Is(err, e) {
			code := clf.ConflictCode
			if code == "" {
				code = "value_already_exist"
			}
			details := gin.H{}
			if clf.ConflictDetails != nil {
				if m := clf.ConflictDetails(err); m != nil {
					details = m
				}
			}
			RespondError(c, http.StatusUnprocessableEntity, code, details)
			return
		}
	}
	if clf.IsTransitionError != nil && clf.IsTransitionError(err) {
		code := clf.TransitionCode
		if code == "" {
			code = "transition_error"
		}
		RespondError(c, http.StatusUnprocessableEntity, code, gin.H{"message": err.Error()})
		return
	}
	for _, e := range clf.TransitionErrors {
		if errors.Is(err, e) {
			code := clf.TransitionCode
			if code == "" {
				code = "transition_error"
			}
			RespondError(c, http.StatusUnprocessableEntity, code, gin.H{"message": err.Error()})
			return
		}
	}
	if clf.IsValidationErr != nil && clf.IsValidationErr(err) {
		RespondError(c, http.StatusUnprocessableEntity, "validation_error", gin.H{"message": err.Error()})
		return
	}
	RespondError(c, http.StatusInternalServerError, "internal_error", gin.H{})
}
