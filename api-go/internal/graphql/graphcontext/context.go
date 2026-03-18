package graphcontext

import "context"

type contextKey string

const (
	organizationIDKey contextKey = "graphql.organization_id"
	apiKeyIDKey       contextKey = "graphql.api_key_id"
)

func WithOrganizationID(ctx context.Context, organizationID string) context.Context {
	return context.WithValue(ctx, organizationIDKey, organizationID)
}

func OrganizationIDFromContext(ctx context.Context) (string, bool) {
	organizationID, ok := ctx.Value(organizationIDKey).(string)
	if !ok || organizationID == "" {
		return "", false
	}
	return organizationID, true
}

func WithAPIKeyID(ctx context.Context, apiKeyID string) context.Context {
	return context.WithValue(ctx, apiKeyIDKey, apiKeyID)
}

func APIKeyIDFromContext(ctx context.Context) (string, bool) {
	apiKeyID, ok := ctx.Value(apiKeyIDKey).(string)
	if !ok || apiKeyID == "" {
		return "", false
	}
	return apiKeyID, true
}
