package types

type (
	SubscribeRequest struct {
		Flag    string
		Token   string
		Type    string
		Profile string
		Preset  string
		UA      string
		Params  map[string]string
	}
	SubscribeResponse struct {
		Config []byte
		Header string
	}
)
