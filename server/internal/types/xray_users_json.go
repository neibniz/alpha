package types

import "encoding/json"

func (r QueryServerXrayUsersResponse) MarshalJSON() ([]byte, error) {
	if !r.Full {
		return json.Marshal(struct {
			Version string `json:"version"`
			Full    bool   `json:"full"`
		}{
			Version: r.Version,
			Full:    r.Full,
		})
	}
	users := r.Users
	if users == nil {
		users = []ServerXrayUser{}
	}
	return json.Marshal(struct {
		Version string           `json:"version"`
		Full    bool             `json:"full"`
		Users   []ServerXrayUser `json:"users"`
	}{
		Version: r.Version,
		Full:    r.Full,
		Users:   users,
	})
}
