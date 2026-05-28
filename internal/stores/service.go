package stores

type Service struct {
	repo *Repo
}

func NewService(repo *Repo) *Service {
	return &Service{repo: repo}
}

func (s *Service) List() []Store {
	return s.repo.List()
}

func (s *Service) Get(id string) (Store, error) {
	return s.repo.Get(id)
}
