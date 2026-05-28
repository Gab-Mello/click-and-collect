package stores

type Store struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Address  string `json:"address"`
	City     string `json:"city"`
	State    string `json:"state"`
	ZIP      string `json:"zip"`
	Capacity int    `json:"capacity"`
	Active   bool   `json:"active"`
}
