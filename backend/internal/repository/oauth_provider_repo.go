package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/oauthprovider"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type oauthProviderRepository struct {
	client *dbent.Client
}

// NewOAuthProviderRepository creates a new OAuth provider repository
func NewOAuthProviderRepository(client *dbent.Client) service.OAuthProviderRepository {
	return &oauthProviderRepository{client: client}
}

func (r *oauthProviderRepository) GetByName(ctx context.Context, name string) (*service.OAuthProvider, error) {
	m, err := r.client.OAuthProvider.Query().
		Where(oauthprovider.NameEQ(name)).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, service.ErrOAuthProviderNotFound
		}
		return nil, err
	}
	return oauthProviderEntityToService(m), nil
}

func (r *oauthProviderRepository) List(ctx context.Context) ([]service.OAuthProvider, error) {
	providers, err := r.client.OAuthProvider.Query().
		Order(dbent.Asc(oauthprovider.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return oauthProviderEntitiesToService(providers), nil
}

func (r *oauthProviderRepository) ListEnabled(ctx context.Context) ([]service.OAuthProvider, error) {
	providers, err := r.client.OAuthProvider.Query().
		Where(oauthprovider.EnabledEQ(true)).
		Order(dbent.Asc(oauthprovider.FieldName)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return oauthProviderEntitiesToService(providers), nil
}

func (r *oauthProviderRepository) Update(ctx context.Context, provider *service.OAuthProvider) error {
	client := clientFromContext(ctx, r.client)
	builder := client.OAuthProvider.UpdateOneID(provider.ID).
		SetClientID(provider.ClientID).
		SetClientSecret(provider.ClientSecret).
		SetEnabled(provider.Enabled)

	if provider.Config != nil {
		builder.SetConfig(provider.Config)
	}

	updated, err := builder.Save(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return service.ErrOAuthProviderNotFound
		}
		return err
	}

	provider.UpdatedAt = updated.UpdatedAt
	return nil
}

// Entity to Service conversions

func oauthProviderEntityToService(m *dbent.OAuthProvider) *service.OAuthProvider {
	if m == nil {
		return nil
	}
	return &service.OAuthProvider{
		ID:           m.ID,
		Name:         m.Name,
		DisplayName:  m.DisplayName,
		ClientID:     m.ClientID,
		ClientSecret: m.ClientSecret,
		Enabled:      m.Enabled,
		Config:       m.Config,
		CreatedAt:    m.CreatedAt,
		UpdatedAt:    m.UpdatedAt,
	}
}

func oauthProviderEntitiesToService(models []*dbent.OAuthProvider) []service.OAuthProvider {
	out := make([]service.OAuthProvider, 0, len(models))
	for i := range models {
		if s := oauthProviderEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
