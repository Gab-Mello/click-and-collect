package stores

import "context"

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) List(ctx context.Context) ([]Store, error) {
	return s.repo.List(ctx)
}

func (s *Service) Get(ctx context.Context, id string) (Store, error) {
	return s.repo.Get(ctx, id)
}
