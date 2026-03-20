package users

import (
	"context"

	"XTalk/services/user/application/users/create_user"
	"XTalk/services/user/application/users/delete_user"
	"XTalk/services/user/application/users/get_user"
	"XTalk/services/user/application/users/get_user_by_email"
	"XTalk/services/user/application/users/update_status"
	"XTalk/services/user/application/users/update_user"
	domainusers "XTalk/services/user/domain/users"
)

// TODO this part should be probably moved

// Service is the application's cohesive use-case facade. Transports depend on
// this API instead of knowing how individual command handlers are assembled.
type Service struct {
	createUser     *create_user.Handler
	deleteUser     *delete_user.Handler
	getUser        *get_user.Handler
	getUserByEmail *get_user_by_email.Handler
	updateStatus   *update_status.Handler
	updateUser     *update_user.Handler
}

func NewService(repository domainusers.UserRepository) *Service {
	return &Service{
		createUser:     create_user.NewHandler(repository),
		deleteUser:     delete_user.NewHandler(repository),
		getUser:        get_user.NewHandler(repository),
		getUserByEmail: get_user_by_email.NewHandler(repository),
		updateStatus:   update_status.NewHandler(repository),
		updateUser:     update_user.NewHandler(repository),
	}
}

func (s *Service) Create(ctx context.Context, command create_user.Command) error {
	return s.createUser.Handle(ctx, command)
}

func (s *Service) Delete(ctx context.Context, command delete_user.Command) error {
	return s.deleteUser.Handle(ctx, command)
}

func (s *Service) Get(ctx context.Context, query get_user.Query) (*get_user.Response, error) {
	return s.getUser.Handle(ctx, query)
}

func (s *Service) GetByEmail(ctx context.Context, query get_user_by_email.Query) (*get_user_by_email.Response, error) {
	return s.getUserByEmail.Handle(ctx, query)
}

func (s *Service) UpdateStatus(ctx context.Context, command update_status.Command) (*update_status.Result, error) {
	return s.updateStatus.Handle(ctx, command)
}

func (s *Service) Update(ctx context.Context, command update_user.Command) (*update_user.Result, error) {
	return s.updateUser.Handle(ctx, command)
}
