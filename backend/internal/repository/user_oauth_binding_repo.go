package repository

import (
	"context"

	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/ent/useroauthbinding"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

type userOAuthBindingRepository struct {
	client *dbent.Client
}

// NewUserOAuthBindingRepository creates a new user OAuth binding repository
func NewUserOAuthBindingRepository(client *dbent.Client) service.UserOAuthBindingRepository {
	return &userOAuthBindingRepository{client: client}
}

func (r *userOAuthBindingRepository) GetByProviderUserID(ctx context.Context, provider, providerUserID string) (*service.UserOAuthBinding, error) {
	m, err := r.client.UserOAuthBinding.Query().
		Where(
			useroauthbinding.ProviderEQ(provider),
			useroauthbinding.ProviderUserIDEQ(providerUserID),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil // Not found is not an error in this case
		}
		return nil, err
	}
	return userOAuthBindingEntityToService(m), nil
}

func (r *userOAuthBindingRepository) GetByUserIDAndProvider(ctx context.Context, userID int64, provider string) (*service.UserOAuthBinding, error) {
	m, err := r.client.UserOAuthBinding.Query().
		Where(
			useroauthbinding.UserIDEQ(userID),
			useroauthbinding.ProviderEQ(provider),
		).
		Only(ctx)
	if err != nil {
		if dbent.IsNotFound(err) {
			return nil, nil // Not found is not an error in this case
		}
		return nil, err
	}
	return userOAuthBindingEntityToService(m), nil
}

func (r *userOAuthBindingRepository) GetByUserID(ctx context.Context, userID int64) ([]service.UserOAuthBinding, error) {
	bindings, err := r.client.UserOAuthBinding.Query().
		Where(useroauthbinding.UserIDEQ(userID)).
		Order(dbent.Asc(useroauthbinding.FieldProvider)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return userOAuthBindingEntitiesToService(bindings), nil
}

func (r *userOAuthBindingRepository) Create(ctx context.Context, binding *service.UserOAuthBinding) error {
	client := clientFromContext(ctx, r.client)
	builder := client.UserOAuthBinding.Create().
		SetUserID(binding.UserID).
		SetProvider(binding.Provider).
		SetProviderUserID(binding.ProviderUserID)

	if binding.ProviderEmail != nil {
		builder.SetProviderEmail(*binding.ProviderEmail)
	}
	if binding.ProviderUsername != nil {
		builder.SetProviderUsername(*binding.ProviderUsername)
	}
	if binding.ProviderAvatar != nil {
		builder.SetProviderAvatar(*binding.ProviderAvatar)
	}
	if binding.AccessToken != nil {
		builder.SetAccessToken(*binding.AccessToken)
	}
	if binding.RefreshToken != nil {
		builder.SetRefreshToken(*binding.RefreshToken)
	}

	created, err := builder.Save(ctx)
	if err != nil {
		return err
	}

	binding.ID = created.ID
	binding.CreatedAt = created.CreatedAt
	binding.UpdatedAt = created.UpdatedAt
	return nil
}

func (r *userOAuthBindingRepository) Delete(ctx context.Context, userID int64, provider string) error {
	client := clientFromContext(ctx, r.client)
	_, err := client.UserOAuthBinding.Delete().
		Where(
			useroauthbinding.UserIDEQ(userID),
			useroauthbinding.ProviderEQ(provider),
		).
		Exec(ctx)
	return err
}

func (r *userOAuthBindingRepository) CountByUserID(ctx context.Context, userID int64) (int, error) {
	return r.client.UserOAuthBinding.Query().
		Where(useroauthbinding.UserIDEQ(userID)).
		Count(ctx)
}

func (r *userOAuthBindingRepository) TransferBinding(ctx context.Context, provider, providerUserID string, newUserID int64, binding *service.UserOAuthBinding) error {
	client := clientFromContext(ctx, r.client)

	// 删除原绑定
	_, err := client.UserOAuthBinding.Delete().
		Where(
			useroauthbinding.ProviderEQ(provider),
			useroauthbinding.ProviderUserIDEQ(providerUserID),
		).
		Exec(ctx)
	if err != nil {
		return err
	}

	// 创建新绑定
	builder := client.UserOAuthBinding.Create().
		SetUserID(newUserID).
		SetProvider(provider).
		SetProviderUserID(providerUserID)

	if binding.ProviderEmail != nil {
		builder.SetProviderEmail(*binding.ProviderEmail)
	}
	if binding.ProviderUsername != nil {
		builder.SetProviderUsername(*binding.ProviderUsername)
	}
	if binding.ProviderAvatar != nil {
		builder.SetProviderAvatar(*binding.ProviderAvatar)
	}
	if binding.AccessToken != nil {
		builder.SetAccessToken(*binding.AccessToken)
	}
	if binding.RefreshToken != nil {
		builder.SetRefreshToken(*binding.RefreshToken)
	}

	_, err = builder.Save(ctx)
	return err
}

// Entity to Service conversions

func userOAuthBindingEntityToService(m *dbent.UserOAuthBinding) *service.UserOAuthBinding {
	if m == nil {
		return nil
	}
	return &service.UserOAuthBinding{
		ID:               m.ID,
		UserID:           m.UserID,
		Provider:         m.Provider,
		ProviderUserID:   m.ProviderUserID,
		ProviderEmail:    m.ProviderEmail,
		ProviderUsername: m.ProviderUsername,
		ProviderAvatar:   m.ProviderAvatar,
		AccessToken:      m.AccessToken,
		RefreshToken:     m.RefreshToken,
		CreatedAt:        m.CreatedAt,
		UpdatedAt:        m.UpdatedAt,
	}
}

func userOAuthBindingEntitiesToService(models []*dbent.UserOAuthBinding) []service.UserOAuthBinding {
	out := make([]service.UserOAuthBinding, 0, len(models))
	for i := range models {
		if s := userOAuthBindingEntityToService(models[i]); s != nil {
			out = append(out, *s)
		}
	}
	return out
}
