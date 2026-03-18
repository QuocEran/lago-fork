package dataloaders

import (
	"context"
	"fmt"
	"time"

	"github.com/graph-gophers/dataloader/v7"
	"gorm.io/gorm"

	"github.com/getlago/lago/api-go/internal/models"
)

type contextKey string

const loaderContextKey contextKey = "graphql.dataloaders"

type Loaders struct {
	OrganizationByID *dataloader.Loader[string, *models.Organization]
}

func NewLoaders(db *gorm.DB) *Loaders {
	return &Loaders{
		OrganizationByID: dataloader.NewBatchedLoader(batchOrganizationsByID(db), dataloader.WithWait[string, *models.Organization](2*time.Millisecond)),
	}
}

func WithLoaders(ctx context.Context, loaders *Loaders) context.Context {
	return context.WithValue(ctx, loaderContextKey, loaders)
}

func ForContext(ctx context.Context) (*Loaders, bool) {
	loaders, ok := ctx.Value(loaderContextKey).(*Loaders)
	if !ok || loaders == nil {
		return nil, false
	}
	return loaders, true
}

func batchOrganizationsByID(db *gorm.DB) func(context.Context, []string) []*dataloader.Result[*models.Organization] {
	return func(ctx context.Context, keys []string) []*dataloader.Result[*models.Organization] {
		results := make([]*dataloader.Result[*models.Organization], len(keys))
		if db == nil {
			err := fmt.Errorf("database is required for organization dataloader")
			for i := range keys {
				results[i] = &dataloader.Result[*models.Organization]{Error: err}
			}
			return results
		}

		organizations := make([]models.Organization, 0, len(keys))
		if err := db.WithContext(ctx).Where("id IN ?", keys).Find(&organizations).Error; err != nil {
			for i := range keys {
				results[i] = &dataloader.Result[*models.Organization]{Error: err}
			}
			return results
		}

		organizationByID := make(map[string]*models.Organization, len(organizations))
		for i := range organizations {
			organization := organizations[i]
			organizationByID[organization.ID] = &organization
		}

		for i, key := range keys {
			organization, exists := organizationByID[key]
			if !exists {
				results[i] = &dataloader.Result[*models.Organization]{Error: gorm.ErrRecordNotFound}
				continue
			}
			results[i] = &dataloader.Result[*models.Organization]{Data: organization}
		}

		return results
	}
}
