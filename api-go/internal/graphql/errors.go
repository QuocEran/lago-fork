package graphql

import (
	"context"
	"errors"

	gqlgraphql "github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

var ErrUnauthorized = errors.New("unauthorized")

func ErrorPresenter(ctx context.Context, err error) *gqlerror.Error {
	formattedError := gqlgraphql.DefaultErrorPresenter(ctx, err)
	if formattedError.Extensions == nil {
		formattedError.Extensions = make(map[string]any)
	}

	if errors.Is(err, ErrUnauthorized) {
		formattedError.Extensions["code"] = "UNAUTHORIZED"
		return formattedError
	}

	if _, exists := formattedError.Extensions["code"]; !exists {
		formattedError.Extensions["code"] = "INTERNAL_SERVER_ERROR"
	}

	return formattedError
}
