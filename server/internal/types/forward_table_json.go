package types

import "encoding/json"

func (r QueryServerForwardTableResponse) MarshalJSON() ([]byte, error) {
	type response QueryServerForwardTableResponse
	out := response(r)
	if out.Items == nil {
		out.Items = []ServerForwardItem{}
	}
	if out.Warnings == nil {
		out.Warnings = []string{}
	}
	return json.Marshal(out)
}
