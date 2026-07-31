package application

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/neibniz/alpha/server/internal/model/client"
	"github.com/neibniz/alpha/server/internal/types"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestSubscribeApplicationMapperUsesExplicitDownloadLinkAndTimestamps(t *testing.T) {
	link := types.DownloadLink{
		IOS:     "ios",
		Android: "android",
		Windows: "windows",
		Mac:     "mac",
		Linux:   "linux",
		Harmony: "harmony",
	}
	modelLink := downloadLinkToModel(link)
	if modelLink.IOS != link.IOS || modelLink.Harmony != link.Harmony {
		t.Fatalf("download link model mapping mismatch: %+v", modelLink)
	}

	createdAt := time.UnixMilli(1716192000000)
	updatedAt := time.UnixMilli(1716195600000)
	got := subscribeApplicationFromModel(&client.SubscribeApplication{
		Id:                9,
		Name:              "Alpha Android",
		Description:       "desc",
		Icon:              "icon",
		Scheme:            "alpha://",
		UserAgent:         "AlphaAgent",
		Enabled:           true,
		IsDefault:         true,
		SubscribeTemplate: "tpl",
		OutputFormat:      "json",
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}, link)

	if got.Id != 9 || got.Name != "Alpha Android" || got.DownloadLink.Android != "android" {
		t.Fatalf("subscribe application mapping mismatch: %+v", got)
	}
	if got.CreatedAt != createdAt.UnixMilli() || got.UpdatedAt != updatedAt.UnixMilli() {
		t.Fatalf("timestamp mapping mismatch: %+v", got)
	}
}

func TestSubscribeApplicationRowsSkipsNilAndKeepsEmptySlice(t *testing.T) {
	got := subscribeApplicationRows([]*client.SubscribeApplication{
		nil,
		{
			Id:           7,
			Name:         "Alpha",
			DownloadLink: `{"ios":" ios ","android":"android"}`,
		},
		{
			Id:           8,
			Name:         "Alpha Desktop",
			DownloadLink: ` {"ios":" ios ","android":"android"} `,
		},
	})
	if len(got) != 2 {
		t.Fatalf("subscribe application rows len = %d, want 2", len(got))
	}
	if got[0].Id != 7 || got[0].DownloadLink.IOS != "ios" || got[0].DownloadLink.Android != "android" {
		t.Fatalf("subscribe application row mismatch: %+v", got[0])
	}
	if got[1].Id != 8 || got[1].DownloadLink.IOS != "ios" || got[1].DownloadLink.Android != "android" {
		t.Fatalf("subscribe application duplicate row mismatch: %+v", got[1])
	}

	empty := subscribeApplicationRows(nil)
	if empty == nil {
		t.Fatal("empty subscribe application rows must return an empty slice, not nil")
	}
	payload, err := json.Marshal(struct {
		List []types.SubscribeApplication `json:"list"`
	}{List: empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(payload) != `{"list":[]}` {
		t.Fatalf("empty subscribe application rows marshal = %s, want list []", payload)
	}
}

func TestSubscribeApplicationListRowsOmitTemplatePayload(t *testing.T) {
	got := subscribeApplicationListRows([]*client.SubscribeApplication{
		nil,
		{
			Id:                7,
			Name:              "Alpha",
			SubscribeTemplate: "large-template",
			DownloadLink:      `{"ios":"ios"}`,
		},
	})
	if len(got) != 1 {
		t.Fatalf("subscribe application list rows len = %d, want 1", len(got))
	}
	if got[0].Id != 7 || got[0].Name != "Alpha" || got[0].DownloadLink.IOS != "ios" {
		t.Fatalf("subscribe application list row mismatch: %+v", got[0])
	}
	payload, err := json.Marshal(struct {
		List []types.SubscribeApplicationListItem `json:"list"`
	}{List: got})
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string][]map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatal(err)
	}
	if _, ok := decoded["list"][0]["template"]; ok {
		t.Fatalf("subscribe application list row must not expose template: %s", payload)
	}
}

func TestSubscribeApplicationRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*client.SubscribeApplication, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &client.SubscribeApplication{Id: 1, Name: "first"}
	rows[len(rows)-1] = &client.SubscribeApplication{Id: 2, Name: "second"}

	got := subscribeApplicationRows(rows)
	if len(got) != 2 {
		t.Fatalf("subscribe application rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy subscribe application rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("subscribe application row order changed: %#v", got)
	}
}

func TestSubscribeApplicationListRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*client.SubscribeApplication, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &client.SubscribeApplication{Id: 1, Name: "first"}
	rows[len(rows)-1] = &client.SubscribeApplication{Id: 2, Name: "second"}

	got := subscribeApplicationListRows(rows)
	if len(got) != 2 {
		t.Fatalf("subscribe application list rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy subscribe application list rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].Id != 1 || got[1].Id != 2 {
		t.Fatalf("subscribe application list row order changed: %#v", got)
	}
}

func TestSubscribeApplicationRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*client.SubscribeApplication, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &client.SubscribeApplication{Id: int64(index + 1), Name: "client"}
	}

	got := subscribeApplicationRows(rows)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("subscribe application rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("subscribe application rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("subscribe application row order changed: %#v", got)
	}
}

func TestDownloadLinkFromRawHandlesEmptyAndInvalidJSON(t *testing.T) {
	if got := downloadLinkFromRaw(""); got != (types.DownloadLink{}) {
		t.Fatalf("empty raw link should return zero value, got %+v", got)
	}
	if got := downloadLinkFromRaw("{invalid"); got != (types.DownloadLink{}) {
		t.Fatalf("invalid raw link should return zero value, got %+v", got)
	}
	got := downloadLinkFromRaw(`{"ios":"ios","linux":"linux"}`)
	if got.IOS != "ios" || got.Linux != "linux" {
		t.Fatalf("raw link mapping mismatch: %+v", got)
	}
}

func TestDownloadLinkToModelTrimsAndOmitsEmptyFields(t *testing.T) {
	modelLink := downloadLinkToModel(types.DownloadLink{
		IOS:     " https://example.com/ios ",
		Android: " ",
		Linux:   "\thttps://example.com/linux\n",
	})
	if modelLink.IOS != "https://example.com/ios" || modelLink.Android != "" || modelLink.Linux != "https://example.com/linux" {
		t.Fatalf("download link was not normalized: %+v", modelLink)
	}

	data, err := modelLink.Marshal()
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"ios":"https://example.com/ios","linux":"https://example.com/linux"}`; got != want {
		t.Fatalf("download link json = %q, want %q", got, want)
	}
}
