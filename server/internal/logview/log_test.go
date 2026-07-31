package logview

import (
	"encoding/json"
	"testing"
	"time"

	modelLog "github.com/neibniz/alpha/server/internal/model/log"
	"github.com/neibniz/alpha/server/internal/viewutil"
)

func TestMessageRowsMasksSensitiveContent(t *testing.T) {
	got := MessageRows([]modelLog.MessageLogRow{
		{
			Id:        10,
			Type:      2,
			Platform:  "email",
			To:        "user@example.com",
			Subject:   "Verify",
			Content:   `{"code":123456,"nested":{"ok":true}}`,
			Status:    1,
			CreatedAt: time.Unix(100, 2000000),
		},
	}, nil, "[test]")

	if len(got) != 1 {
		t.Fatalf("expected 1 row, got %d", len(got))
	}
	if got[0].To != "u**r@example.com" {
		t.Fatalf("unexpected masked recipient: %s", got[0].To)
	}
	if string(got[0].Content) != `{"code":"***","nested":{"ok":true}}` {
		t.Fatalf("unexpected masked content: %s", got[0].Content)
	}

	payload, err := json.Marshal(got[0])
	if err != nil {
		t.Fatalf("marshal message log: %v", err)
	}
	var decoded struct {
		Content map[string]any `json:"content"`
	}
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal marshaled message log: %v", err)
	}
	if decoded.Content["code"].(string) != "***" {
		t.Fatalf("unexpected marshaled content: %s", payload)
	}
}

func TestAdminRowsReturnCompleteFieldsOnlyWhenRevealed(t *testing.T) {
	login := LoginRows([]modelLog.LoginLogRow{{
		LoginIP:   "203.0.113.42",
		UserAgent: "AlphaClient/1.0 private-details",
		Revealed:  true,
	}}, nil)
	if login[0].LoginIP != "203.0.113.42" || login[0].UserAgent != "AlphaClient/1.0 private-details" || !login[0].Revealed {
		t.Fatalf("revealed login row was masked: %#v", login[0])
	}

	register := RegisterRows([]modelLog.RegisterLogRow{{
		Identifier: "user@example.com",
		RegisterIP: "198.51.100.8",
		UserAgent:  "AlphaRegister/1.0 private-details",
		Revealed:   true,
	}}, nil)
	if register[0].Identifier != "user@example.com" || register[0].RegisterIP != "198.51.100.8" || !register[0].Revealed {
		t.Fatalf("revealed register row was masked: %#v", register[0])
	}

	message := MessageRows([]modelLog.MessageLogRow{{
		To:       "user@example.com",
		Content:  `{"code":123456}`,
		Revealed: true,
	}}, nil, "[test]")
	if message[0].To != "user@example.com" || string(message[0].Content) != `{"code":123456}` || !message[0].Revealed {
		t.Fatalf("revealed message row was masked: %#v", message[0])
	}

	subscribe := SubscribeRows([]*modelLog.SubscribeAccessLog{{
		ClientIP:  "192.0.2.5",
		UserAgent: "Clash/2.0 private-details",
		Revealed:  true,
	}}, nil)
	if subscribe[0].ClientIP != "192.0.2.5" || subscribe[0].UserAgent != "Clash/2.0 private-details" || !subscribe[0].Revealed {
		t.Fatalf("revealed subscribe row was masked: %#v", subscribe[0])
	}
}

func TestMessageRowsPreservesRowsWithInvalidContent(t *testing.T) {
	got := MessageRows([]modelLog.MessageLogRow{
		{Id: 11, Content: `{invalid`, CreatedAt: time.Unix(1, 0)},
		{Id: 12, Content: `   `, CreatedAt: time.Unix(2, 0)},
	}, nil, "[test]")

	if len(got) != 2 {
		t.Fatalf("invalid content should not drop rows, got %d", len(got))
	}
	for _, item := range got {
		if string(item.Content) != "null" {
			t.Fatalf("invalid or empty content should marshal as null, got %s", item.Content)
		}
	}
}

func TestMessageRowsReusesDuplicateMaskedContent(t *testing.T) {
	content := `{"code":123456,"nested":{"ok":true}}`
	got := MessageRows([]modelLog.MessageLogRow{
		{Id: 11, Content: content},
		{Id: 12, Content: content},
	}, nil, "[test]")

	if len(got) != 2 {
		t.Fatalf("message rows len = %d, want 2", len(got))
	}
	if string(got[0].Content) != `{"code":"***","nested":{"ok":true}}` || string(got[1].Content) != string(got[0].Content) {
		t.Fatalf("masked content mismatch: %#v", got)
	}
	if len(got[0].Content) == 0 || &got[0].Content[0] != &got[1].Content[0] {
		t.Fatalf("duplicate content should reuse masked raw message storage")
	}
}

func TestMessageContentCacheSkipsInvalidContent(t *testing.T) {
	cache := newMessageContentCache(2)
	first := cache.Content(`{invalid`, nil, "[test]", 11)
	second := cache.Content(`{invalid`, nil, "[test]", 12)
	if string(first) != "null" || string(second) != "null" {
		t.Fatalf("invalid content should stay null: %s %s", first, second)
	}
	if len(cache.rows) != 0 {
		t.Fatalf("invalid content should not populate cache: %#v", cache.rows)
	}
}

func TestMessageContentCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newMessageContentCache(smallMessageContentCacheThreshold + 1)
	if cache == nil || cache.index != nil || cache.rows != nil {
		t.Fatalf("large message content cache should not preallocate storage from hint alone: %#v", cache)
	}
	first := cache.Content(`{"code":123456}`, nil, "[test]", 1)
	second := cache.Content(`{"code":123456}`, nil, "[test]", 2)
	if string(first) != string(second) || len(cache.rows) != 1 || cache.index != nil {
		t.Fatalf("duplicate content should stay on the small linear path: rows=%#v index=%#v", cache.rows, cache.index)
	}

	cache = newMessageContentCache(smallMessageContentCacheThreshold + 1)
	for i := 0; i <= smallMessageContentCacheThreshold; i++ {
		raw := `{"code":` + string(rune('1'+i)) + `}`
		cache.Content(raw, nil, "[test]", int64(i))
	}
	if len(cache.rows) != smallMessageContentCacheThreshold+1 || len(cache.index) != smallMessageContentCacheThreshold+1 {
		t.Fatalf("message content cache rows/index = %d/%d", len(cache.rows), len(cache.index))
	}
}

func TestMessageContentCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newMessageContentCache(smallMessageContentCacheThreshold + 20)
	for i := 0; i <= smallMessageContentCacheThreshold; i++ {
		raw := `{"code":` + string(rune('1'+i)) + `}`
		cache.Content(raw, nil, "[test]", int64(i))
	}
	if cache.index == nil {
		t.Fatal("message content cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("message content cache row cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
	for i := 0; i < smallMessageContentCacheThreshold+20; i++ {
		cache.Content(`{"code":1}`, nil, "[test]", int64(i))
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("message content cache row cap after duplicates = %d, want at most %d", cap(cache.rows), maxCap)
	}
}

func TestMessageContentKeepsLiteralNull(t *testing.T) {
	got := MessageContent(" null ", nil, "[test]", 13)
	if string(got) != "null" {
		t.Fatalf("literal null content should stay null, got %s", got)
	}
}

func TestUserSubscribeRowsSkipsNilAndMasksPrivateFields(t *testing.T) {
	rows := []*modelLog.SubscribeAccessLog{
		nil,
		{
			Id:              7,
			UserId:          3,
			UserSubscribeId: 9,
			TokenPrefix:     "abcdef123456",
			ClientIP:        "203.0.113.9",
			CountryShort:    "US",
			Region:          "CO",
			UserAgent:       "AlphaClient/1.2.3",
			CreatedAt:       time.Unix(20, 0),
		},
	}
	got := UserSubscribeRows(rows, func(ip, countryShort, region string) Location {
		return Location{CountryShort: countryShort, Region: region, Location: countryShort + " " + region}
	})

	if len(got) != 1 {
		t.Fatalf("expected nil row to be skipped, got %d rows", len(got))
	}
	if got[0].Token == "abcdef123456" || got[0].IP == "203.0.113.9" || got[0].UserAgent == "AlphaClient/1.2.3" {
		t.Fatalf("private fields were not masked: %#v", got[0])
	}
	if got[0].Location != "US CO" || got[0].Timestamp != 20000 {
		t.Fatalf("unexpected mapped row: %#v", got[0])
	}
}

func TestSubscribeRowsSkipNilWithoutInputSizedCapacity(t *testing.T) {
	rows := make([]*modelLog.SubscribeAccessLog, viewutil.SmallRowsCapacityThreshold+20)
	rows[0] = &modelLog.SubscribeAccessLog{UserId: 1, ClientIP: "203.0.113.1"}
	rows[len(rows)-1] = &modelLog.SubscribeAccessLog{UserId: 2, ClientIP: "203.0.113.2"}

	got := SubscribeRows(rows, nil)
	if len(got) != 2 {
		t.Fatalf("subscribe rows len = %d, want 2", len(got))
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("nil-heavy subscribe rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[0].UserId != 1 || got[1].UserId != 2 {
		t.Fatalf("subscribe row order changed: %#v", got)
	}
}

func TestUserSubscribeRowsUseCapacityHintAfterActualThreshold(t *testing.T) {
	rows := make([]*modelLog.SubscribeAccessLog, viewutil.SmallRowsCapacityThreshold+20)
	for index := 0; index <= viewutil.SmallRowsCapacityThreshold; index++ {
		rows[index] = &modelLog.SubscribeAccessLog{Id: int64(index + 1), ClientIP: "203.0.113.1"}
	}

	got := UserSubscribeRows(rows, nil)
	if len(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("user subscribe rows len = %d, want %d", len(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if cap(got) != viewutil.SmallRowsCapacityThreshold+1 {
		t.Fatalf("user subscribe rows cap = %d, want %d", cap(got), viewutil.SmallRowsCapacityThreshold+1)
	}
	if got[len(got)-1].Id != int64(viewutil.SmallRowsCapacityThreshold+1) {
		t.Fatalf("user subscribe row order changed: %#v", got)
	}
}

func TestGiftRowsMapsUserIDAndKeepsJSONShape(t *testing.T) {
	got := GiftRows([]modelLog.GiftLogRow{{
		UserId:      42,
		Type:        3,
		OrderNo:     "order-1",
		SubscribeId: 9,
		Amount:      100,
		Balance:     900,
		Remark:      "manual gift",
		Timestamp:   1700000000,
	}})
	if len(got) != 1 {
		t.Fatalf("gift rows len = %d, want 1", len(got))
	}
	if got[0].UserId != 42 || got[0].SubscribeId != 9 || got[0].Amount != 100 || got[0].Balance != 900 {
		t.Fatalf("gift row fields not mapped: %#v", got[0])
	}

	payload, err := json.Marshal(got[0])
	if err != nil {
		t.Fatalf("marshal gift row: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal gift row: %v", err)
	}
	if decoded["user_id"] != float64(42) {
		t.Fatalf("gift row JSON missing user_id: %s", payload)
	}
}

func TestRowsPreferStoredLocationBeforeResolver(t *testing.T) {
	calls := 0
	resolve := func(ip, countryShort, region string) Location {
		calls++
		return Location{CountryShort: "ZZ", Region: "Resolver", Location: "Resolver Location"}
	}

	loginRows := LoginRows([]modelLog.LoginLogRow{{
		LoginIP:      "203.0.113.10",
		CountryShort: "US",
		Region:       "CA",
		Location:     "Stored Location",
	}}, resolve)
	if len(loginRows) != 1 || loginRows[0].Location != "Stored Location" || loginRows[0].CountryShort != "US" || loginRows[0].Region != "CA" {
		t.Fatalf("stored login location not preserved: %#v", loginRows)
	}

	subscribeRows := SubscribeRows([]*modelLog.SubscribeAccessLog{{
		ClientIP:     "203.0.113.11",
		CountryShort: "JP",
		Region:       "13",
		Location:     "Tokyo",
	}}, resolve)
	if len(subscribeRows) != 1 || subscribeRows[0].Location != "Tokyo" || subscribeRows[0].CountryShort != "JP" || subscribeRows[0].Region != "13" {
		t.Fatalf("stored subscribe location not preserved: %#v", subscribeRows)
	}

	if calls != 0 {
		t.Fatalf("resolver calls = %d, want 0 when stored location is present", calls)
	}
}

func TestRowsCacheResolvedLocationsWithinResponse(t *testing.T) {
	calls := 0
	resolve := func(ip, countryShort, region string) Location {
		calls++
		return Location{CountryShort: countryShort, Region: region, Location: "resolved:" + ip}
	}

	got := LoginRows([]modelLog.LoginLogRow{
		{LoginIP: "203.0.113.10", CountryShort: "US", Region: "CO"},
		{LoginIP: "203.0.113.10", CountryShort: " US ", Region: " CO "},
		{LoginIP: "203.0.113.11", CountryShort: "US", Region: "CO"},
	}, resolve)
	if len(got) != 3 {
		t.Fatalf("login rows len = %d, want 3", len(got))
	}
	if calls != 2 {
		t.Fatalf("resolver calls = %d, want 2 for two distinct location keys", calls)
	}
	if got[0].Location != "resolved:203.0.113.10" || got[1].Location != got[0].Location || got[2].Location != "resolved:203.0.113.11" {
		t.Fatalf("cached locations not reused correctly: %#v", got)
	}
}

func TestLocationResolverCacheUsesLinearPathForSmallInput(t *testing.T) {
	calls := 0
	cache := newLocationResolverCache(func(ip, countryShort, region string) Location {
		calls++
		return Location{CountryShort: countryShort, Region: region, Location: ip}
	})

	first := cache.Resolve("203.0.113.10", "US", "CO", "")
	second := cache.Resolve("203.0.113.10", " US ", " CO ", "")
	if cache.index != nil {
		t.Fatal("small location resolver cache should not allocate map")
	}
	if calls != 1 {
		t.Fatalf("resolver calls = %d, want 1", calls)
	}
	if first != second || first.Location != "203.0.113.10" {
		t.Fatalf("cached location mismatch: first=%#v second=%#v", first, second)
	}
}

func TestLocationResolverCacheUpgradesLargeInputToMap(t *testing.T) {
	cache := newLocationResolverCache(func(ip, countryShort, region string) Location {
		return Location{CountryShort: countryShort, Region: region, Location: ip}
	})
	for i := 0; i <= smallLocationCacheThreshold; i++ {
		cache.Resolve("203.0.113."+string(rune('a'+i)), "US", "CO", "")
	}
	if cache.index == nil {
		t.Fatal("large location resolver cache should allocate map")
	}
	if len(cache.rows) != smallLocationCacheThreshold+1 {
		t.Fatalf("location cache rows len = %d, want %d", len(cache.rows), smallLocationCacheThreshold+1)
	}
}

func TestLocationResolverCacheUsesCapacityHintLazily(t *testing.T) {
	cache := newLocationResolverCacheWithHint(func(ip, countryShort, region string) Location {
		return Location{CountryShort: countryShort, Region: region, Location: ip}
	}, smallLocationCacheThreshold+1)
	if cache.rows != nil || cache.index != nil {
		t.Fatalf("hinted location cache should not allocate before first resolve: %#v", cache)
	}

	first := cache.Resolve("203.0.113.10", "US", "CO", "")
	second := cache.Resolve("203.0.113.10", " US ", " CO ", "")
	if cap(cache.rows) != smallLocationCacheThreshold+1 {
		t.Fatalf("location cache rows cap = %d, want %d", cap(cache.rows), smallLocationCacheThreshold+1)
	}
	if len(cache.rows) != 1 || cache.index != nil || first != second {
		t.Fatalf("location cache should keep one cached entry: rows=%#v first=%#v second=%#v", cache.rows, first, second)
	}
}

func TestLocationResolverCacheBoundsSparseUpgradeCapacity(t *testing.T) {
	cache := newLocationResolverCacheWithHint(func(ip, countryShort, region string) Location {
		return Location{CountryShort: countryShort, Region: region, Location: ip}
	}, smallLocationCacheThreshold+20)
	for i := 0; i <= smallLocationCacheThreshold; i++ {
		cache.Resolve("203.0.113."+string(rune('a'+i)), "US", "CO", "")
	}
	if cache.index == nil {
		t.Fatal("location cache should allocate map after actual unique values exceed threshold")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("location cache rows cap = %d, want at most %d", cap(cache.rows), maxCap)
	}
	for i := 0; i < smallLocationCacheThreshold+20; i++ {
		cache.Resolve("203.0.113.a", "US", "CO", "")
	}
	if maxCap := len(cache.rows) * 2; cap(cache.rows) > maxCap {
		t.Fatalf("location cache rows cap after duplicates = %d, want at most %d", cap(cache.rows), maxCap)
	}
}

func TestLocationResolverCacheDoesNotAllocateForStoredLocation(t *testing.T) {
	cache := newLocationResolverCacheWithHint(func(ip, countryShort, region string) Location {
		t.Fatal("resolver should not be called when stored location exists")
		return Location{}
	}, smallLocationCacheThreshold+1)
	location := cache.Resolve("203.0.113.10", " US ", " CO ", " Denver ")
	if location.Location != "Denver" || location.CountryShort != "US" || location.Region != "CO" {
		t.Fatalf("stored location not normalized: %#v", location)
	}
	if cache.rows != nil || cache.index != nil {
		t.Fatalf("stored locations should not populate resolver cache: %#v", cache)
	}
}

func TestRowsKeepStoredLocationWithoutResolver(t *testing.T) {
	registerRows := RegisterRows([]modelLog.RegisterLogRow{{
		CountryShort: " US ",
		Region:       " CO ",
		Location:     " Denver ",
	}}, nil)
	if len(registerRows) != 1 || registerRows[0].Location != "Denver" || registerRows[0].CountryShort != "US" || registerRows[0].Region != "CO" {
		t.Fatalf("stored register location not normalized: %#v", registerRows)
	}

	userSubscribeRows := UserSubscribeRows([]*modelLog.SubscribeAccessLog{{
		CountryShort: " DE ",
		Region:       " BE ",
		Location:     " Berlin ",
	}}, nil)
	if len(userSubscribeRows) != 1 || userSubscribeRows[0].Location != "Berlin" || userSubscribeRows[0].CountryShort != "DE" || userSubscribeRows[0].Region != "BE" {
		t.Fatalf("stored user subscribe location not normalized: %#v", userSubscribeRows)
	}
}

func TestRowsKeepZeroTimesAtZero(t *testing.T) {
	loginRows := LoginRows([]modelLog.LoginLogRow{{}}, nil)
	if len(loginRows) != 1 || loginRows[0].Timestamp != 0 {
		t.Fatalf("zero login time should stay zero: %#v", loginRows)
	}

	userLoginRows := UserLoginRows([]modelLog.LoginLogRow{{}}, nil)
	if len(userLoginRows) != 1 || userLoginRows[0].Timestamp != 0 {
		t.Fatalf("zero user login time should stay zero: %#v", userLoginRows)
	}

	subscribeRows := SubscribeRows([]*modelLog.SubscribeAccessLog{{}}, nil)
	if len(subscribeRows) != 1 || subscribeRows[0].Timestamp != 0 {
		t.Fatalf("zero subscribe time should stay zero: %#v", subscribeRows)
	}

	userSubscribeRows := UserSubscribeRows([]*modelLog.SubscribeAccessLog{{}}, nil)
	if len(userSubscribeRows) != 1 || userSubscribeRows[0].Timestamp != 0 {
		t.Fatalf("zero user subscribe time should stay zero: %#v", userSubscribeRows)
	}

	messageRows := MessageRows([]modelLog.MessageLogRow{{}}, nil, "[test]")
	if len(messageRows) != 1 || messageRows[0].CreatedAt != 0 {
		t.Fatalf("zero message time should stay zero: %#v", messageRows)
	}
}

func TestRowsKeepEmptyListShape(t *testing.T) {
	if rows := LoginRows(nil, nil); rows == nil || len(rows) != 0 {
		t.Fatalf("empty login rows should be non-nil empty slice: %#v", rows)
	}
	if rows := SubscribeRows(nil, nil); rows == nil || len(rows) != 0 {
		t.Fatalf("empty subscribe rows should be non-nil empty slice: %#v", rows)
	}
	if rows := MessageRows(nil, nil, "[test]"); rows == nil || len(rows) != 0 {
		t.Fatalf("empty message rows should be non-nil empty slice: %#v", rows)
	}
}
