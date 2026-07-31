package types

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

var rawMessageType = reflect.TypeOf(json.RawMessage{})
var rawMessageSliceType = reflect.TypeOf([]json.RawMessage{})
var apiContractStructRE = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_]+)\s*\{`)
var generatedContractStructRE = regexp.MustCompile(`(?m)^type\s+([A-Za-z0-9_]+)\s+struct\s*\{`)
var generatedContractAliasRE = regexp.MustCompile(`(?m)^type\s+([A-Za-z0-9_]+)\s*=\s*([A-Za-z0-9_]+)\s*$`)
var contractTagRE = regexp.MustCompile(`(json|form|path):"([^"]+)"`)
var contractEmbeddedTypeRE = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func TestDynamicJSONFieldsUseRawMessage(t *testing.T) {
	tests := []struct {
		name      string
		model     any
		field     string
		fieldType reflect.Type
	}{
		{name: "auth config", model: AuthMethodConfig{}, field: "Config", fieldType: rawMessageType},
		{name: "auth update config", model: UpdateAuthMethodConfigRequest{}, field: "Config", fieldType: rawMessageType},
		{name: "payment create config", model: CreatePaymentMethodRequest{}, field: "Config", fieldType: rawMessageType},
		{name: "payment update config", model: UpdatePaymentMethodRequest{}, field: "Config", fieldType: rawMessageType},
		{name: "payment config", model: PaymentConfig{}, field: "Config", fieldType: rawMessageType},
		{name: "template config", model: XrayTemplate{}, field: "Config", fieldType: rawMessageType},
		{name: "template variables schema", model: XrayTemplate{}, field: "VariablesSchema", fieldType: rawMessageType},
		{name: "template default variables", model: XrayTemplate{}, field: "DefaultVariables", fieldType: rawMessageType},
		{name: "template subscription meta", model: XrayTemplate{}, field: "SubscriptionMeta", fieldType: rawMessageType},
		{name: "template catalog config", model: XrayTemplateCatalogItem{}, field: "Config", fieldType: rawMessageType},
		{name: "template catalog variables schema", model: XrayTemplateCatalogItem{}, field: "VariablesSchema", fieldType: rawMessageType},
		{name: "template catalog default variables", model: XrayTemplateCatalogItem{}, field: "DefaultVariables", fieldType: rawMessageType},
		{name: "template catalog subscription meta", model: XrayTemplateCatalogItem{}, field: "SubscriptionMeta", fieldType: rawMessageType},
		{name: "create template config", model: CreateXrayTemplateRequest{}, field: "Config", fieldType: rawMessageType},
		{name: "create template variables schema", model: CreateXrayTemplateRequest{}, field: "VariablesSchema", fieldType: rawMessageType},
		{name: "create template default variables", model: CreateXrayTemplateRequest{}, field: "DefaultVariables", fieldType: rawMessageType},
		{name: "create template subscription meta", model: CreateXrayTemplateRequest{}, field: "SubscriptionMeta", fieldType: rawMessageType},
		{name: "update template config", model: UpdateXrayTemplateRequest{}, field: "Config", fieldType: rawMessageType},
		{name: "update template variables schema", model: UpdateXrayTemplateRequest{}, field: "VariablesSchema", fieldType: rawMessageType},
		{name: "update template default variables", model: UpdateXrayTemplateRequest{}, field: "DefaultVariables", fieldType: rawMessageType},
		{name: "update template subscription meta", model: UpdateXrayTemplateRequest{}, field: "SubscriptionMeta", fieldType: rawMessageType},
		{name: "server template variables", model: ServerXrayTemplateBinding{}, field: "Variables", fieldType: rawMessageType},
		{name: "server template subscription variables", model: ServerXrayTemplateBinding{}, field: "SubscriptionVariables", fieldType: rawMessageType},
		{name: "server template response variables", model: ServerXrayTemplate{}, field: "Variables", fieldType: rawMessageType},
		{name: "server template response subscription variables", model: ServerXrayTemplate{}, field: "SubscriptionVariables", fieldType: rawMessageType},
		{name: "xray config inbounds", model: QueryServerXrayConfigResponse{}, field: "Inbounds", fieldType: rawMessageSliceType},
		{name: "xray config outbounds", model: QueryServerXrayConfigResponse{}, field: "Outbounds", fieldType: rawMessageSliceType},
		{name: "xray config dns", model: QueryServerXrayConfigResponse{}, field: "DNS", fieldType: rawMessageType},
		{name: "xray config routing", model: QueryServerXrayConfigResponse{}, field: "Routing", fieldType: rawMessageType},
		{name: "xray config geodata", model: QueryServerXrayConfigResponse{}, field: "Geodata", fieldType: rawMessageType},
		{name: "system log content", model: SystemLogEntry{}, field: "Content", fieldType: rawMessageType},
		{name: "system log fields", model: SystemLogEntry{}, field: "Fields", fieldType: rawMessageType},
		{name: "message log content", model: MessageLog{}, field: "Content", fieldType: rawMessageType},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field, ok := reflect.TypeOf(tt.model).FieldByName(tt.field)
			if !ok {
				t.Fatalf("%T.%s field not found", tt.model, tt.field)
			}
			if field.Type != tt.fieldType {
				t.Fatalf("%T.%s type = %s, want %s", tt.model, tt.field, field.Type, tt.fieldType)
			}
		})
	}
}

func TestOAuthCallbackPayloadUsesStructuredFields(t *testing.T) {
	callbackType := reflect.TypeOf(OAuthCallbackPayload{})
	for _, tt := range []struct {
		field string
		tag   string
	}{
		{field: "Code", tag: "code,omitempty"},
		{field: "State", tag: "state,omitempty"},
		{field: "TgAuthResult", tag: "tgAuthResult,omitempty"},
	} {
		field, ok := callbackType.FieldByName(tt.field)
		if !ok {
			t.Fatalf("OAuthCallbackPayload.%s field not found", tt.field)
		}
		if field.Type.Kind() != reflect.String {
			t.Fatalf("OAuthCallbackPayload.%s type = %s, want string", tt.field, field.Type)
		}
		if got := field.Tag.Get("json"); got != tt.tag {
			t.Fatalf("OAuthCallbackPayload.%s json tag = %q, want %q", tt.field, got, tt.tag)
		}
	}

	loginCallback, ok := reflect.TypeOf(OAuthLoginGetTokenRequest{}).FieldByName("Callback")
	if !ok {
		t.Fatal("OAuthLoginGetTokenRequest.Callback field not found")
	}
	if loginCallback.Type != callbackType {
		t.Fatalf("OAuthLoginGetTokenRequest.Callback type = %s, want %s", loginCallback.Type, callbackType)
	}
	bindCallback, ok := reflect.TypeOf(BindOAuthCallbackRequest{}).FieldByName("Callback")
	if !ok {
		t.Fatal("BindOAuthCallbackRequest.Callback field not found")
	}
	if bindCallback.Type != callbackType {
		t.Fatalf("BindOAuthCallbackRequest.Callback type = %s, want %s", bindCallback.Type, callbackType)
	}
}

func TestGetRequestsUseQueryTags(t *testing.T) {
	tests := []struct {
		name   string
		model  any
		fields map[string]string
	}{
		{
			name:  "document detail",
			model: GetDocumentDetailRequest{},
			fields: map[string]string{
				"Id": "id",
			},
		},
		{
			name:  "user auth methods",
			model: GetUserAuthMethodRequest{},
			fields: map[string]string{
				"UserId": "user_id",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reqType := reflect.TypeOf(tt.model)
			for fieldName, want := range tt.fields {
				field, ok := reqType.FieldByName(fieldName)
				if !ok {
					t.Fatalf("%T.%s field not found", tt.model, fieldName)
				}
				if got := field.Tag.Get("form"); got != want {
					t.Fatalf("%T.%s form tag = %q, want %q", tt.model, fieldName, got, want)
				}
				if got := field.Tag.Get("json"); got != "" {
					t.Fatalf("%T.%s must not use JSON binding for GET params, got %q", tt.model, fieldName, got)
				}
			}
		})
	}
}

func TestAPISourceMatchesGeneratedTypes(t *testing.T) {
	tests := []struct {
		name     string
		apiPath  string
		typeName string
		model    any
	}{
		{name: "server status", apiPath: "../../apis/admin/server.api", typeName: "ServerStatus", model: ServerStatus{}},
		{name: "admin server list item", apiPath: "../../apis/admin/server.api", typeName: "ServerListItem", model: ServerListItem{}},
		{name: "admin node list item", apiPath: "../../apis/admin/server.api", typeName: "NodeListItem", model: NodeListItem{}},
		{name: "user", apiPath: "../../apis/types.api", typeName: "User", model: User{}},
		{name: "user list item", apiPath: "../../apis/types.api", typeName: "UserListItem", model: UserListItem{}},
		{name: "membership policy", apiPath: "../../apis/types.api", typeName: "MembershipPolicy", model: MembershipPolicy{}},
		{name: "security config", apiPath: "../../apis/types.api", typeName: "SecurityConfig", model: SecurityConfig{}},
		{name: "admin user subscribe detail", apiPath: "../../apis/admin/user.api", typeName: "UserSubscribeDetail", model: UserSubscribeDetail{}},
		{name: "admin user subscribe product summary", apiPath: "../../apis/types.api", typeName: "UserSubscribeProductSummary", model: UserSubscribeProductSummary{}},
		{name: "admin user subscribe list item", apiPath: "../../apis/types.api", typeName: "UserSubscribeListItem", model: UserSubscribeListItem{}},
		{name: "admin user subscribe access", apiPath: "../../apis/types.api", typeName: "UserSubscribeAccess", model: UserSubscribeAccess{}},
		{name: "public user subscribe info", apiPath: "../../apis/public/subscribe.api", typeName: "UserSubscribeInfo", model: UserSubscribeInfo{}},
		{name: "public user subscribe node", apiPath: "../../apis/public/subscribe.api", typeName: "UserSubscribeNodeInfo", model: UserSubscribeNodeInfo{}},
		{name: "admin ticket list item", apiPath: "../../apis/types.api", typeName: "TicketListItem", model: TicketListItem{}},
		{name: "user ticket list item", apiPath: "../../apis/types.api", typeName: "UserTicketListItem", model: UserTicketListItem{}},
		{name: "auth method summary", apiPath: "../../apis/types.api", typeName: "AuthMethodSummary", model: AuthMethodSummary{}},
		{name: "payment method list item", apiPath: "../../apis/types.api", typeName: "PaymentMethodListItem", model: PaymentMethodListItem{}},
		{name: "payment method summary", apiPath: "../../apis/types.api", typeName: "PaymentMethodSummary", model: PaymentMethodSummary{}},
		{name: "coupon list item", apiPath: "../../apis/types.api", typeName: "CouponListItem", model: CouponListItem{}},
		{name: "admin order list item", apiPath: "../../apis/types.api", typeName: "OrderListItem", model: OrderListItem{}},
		{name: "xray template list item", apiPath: "../../apis/types.api", typeName: "XrayTemplateListItem", model: XrayTemplateListItem{}},
		{name: "xray template catalog item", apiPath: "../../apis/types.api", typeName: "XrayTemplateCatalogItem", model: XrayTemplateCatalogItem{}},
		{name: "admin subscribe list item", apiPath: "../../apis/admin/subscribe.api", typeName: "SubscribeItem", model: SubscribeItem{}},
		{name: "subscribe application list item", apiPath: "../../apis/admin/application.api", typeName: "SubscribeApplicationListItem", model: SubscribeApplicationListItem{}},
		{name: "document list item", apiPath: "../../apis/types.api", typeName: "DocumentListItem", model: DocumentListItem{}},
		{name: "public document list item", apiPath: "../../apis/types.api", typeName: "PublicDocumentListItem", model: PublicDocumentListItem{}},
		{name: "announcement list item", apiPath: "../../apis/types.api", typeName: "AnnouncementListItem", model: AnnouncementListItem{}},
		{name: "ads list item", apiPath: "../../apis/types.api", typeName: "AdsListItem", model: AdsListItem{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			source, err := os.ReadFile(tt.apiPath)
			if err != nil {
				t.Fatalf("read api source: %v", err)
			}
			body := apiTypeBody(t, string(source), tt.typeName)
			got := apiJSONFieldSet(body)
			want := structJSONFieldSet(reflect.TypeOf(tt.model))
			if !reflect.DeepEqual(got, want) {
				t.Fatalf("%s .api JSON fields = %#v, want generated type fields %#v", tt.typeName, got, want)
			}
		})
	}
}

func TestAllAPISourceTypeTagsMatchGeneratedTypes(t *testing.T) {
	apiStructs, apiBodies := parseAPIContractStructs(t, "../../apis")
	generatedBodies := parseGeneratedTypeStructs(t, "types.go")

	for _, apiStruct := range apiStructs {
		t.Run(apiStruct.Name, func(t *testing.T) {
			want := contractTaggedFieldSet(apiStruct.Name, apiBodies, nil)
			if len(want) == 0 {
				return
			}
			if _, ok := generatedBodies[apiStruct.Name]; !ok {
				t.Fatalf("%s is defined in %s but missing from generated types.go", apiStruct.Name, apiStruct.File)
			}
			got := contractTaggedFieldSet(apiStruct.Name, generatedBodies, nil)
			if !reflect.DeepEqual(got, want) {
				t.Fatalf(
					"%s generated field tags = %#v, want .api field tags %#v from %s",
					apiStruct.Name,
					sortedFieldSet(got),
					sortedFieldSet(want),
					apiStruct.File,
				)
			}
		})
	}
}

func TestDocumentListContentOmitsWhenEmpty(t *testing.T) {
	payload, err := json.Marshal(Document{
		Id:    1,
		Title: "guide",
		Tags:  []string{"quickstart"},
		Show:  true,
	})
	if err != nil {
		t.Fatalf("json.Marshal(Document) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(Document) error = %v", err)
	}
	if _, ok := got["content"]; ok {
		t.Fatalf("empty document content must be omitted from list JSON: %s", payload)
	}
}

func TestQueryWithdrawalLogListResponseKeepsEmptyListArray(t *testing.T) {
	payload, err := json.Marshal(QueryWithdrawalLogListResponse{
		List:  []WithdrawalLog{},
		Total: 0,
	})
	if err != nil {
		t.Fatalf("json.Marshal(QueryWithdrawalLogListResponse) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(QueryWithdrawalLogListResponse) error = %v", err)
	}
	list, ok := got["list"].([]any)
	if !ok || len(list) != 0 {
		t.Fatalf("withdrawal list response must include stable empty list array: %s", payload)
	}
	if _, ok := got["total"]; !ok {
		t.Fatalf("withdrawal list response must include total: %s", payload)
	}
}

func TestAuthMethodSummaryKeepsListPayloadLean(t *testing.T) {
	payload, err := json.Marshal(AuthMethodSummary{
		Id:      1,
		Method:  "email",
		Enabled: true,
	})
	if err != nil {
		t.Fatalf("json.Marshal(AuthMethodSummary) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(AuthMethodSummary) error = %v", err)
	}
	if _, ok := got["config"]; ok {
		t.Fatalf("auth method summary must not expose dynamic config: %s", payload)
	}
	for _, key := range []string{"id", "method", "enabled"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("auth method summary missing %q: %s", key, payload)
		}
	}
}

func TestPaymentMethodListItemKeepsListPayloadLean(t *testing.T) {
	payload, err := json.Marshal(PaymentMethodListItem{
		Id:          1,
		Name:        "Stripe",
		Platform:    "stripe",
		Description: "card",
		Icon:        "icon",
		FeeMode:     1,
		FeePercent:  5,
		FeeAmount:   0,
		Enable:      true,
		NotifyURL:   "https://pay.example.com/v1/notify/stripe/token",
	})
	if err != nil {
		t.Fatalf("json.Marshal(PaymentMethodListItem) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(PaymentMethodListItem) error = %v", err)
	}
	for _, key := range []string{"domain", "config"} {
		if _, ok := got[key]; ok {
			t.Fatalf("payment method list item must not expose %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "name", "platform", "description", "icon", "fee_mode", "fee_percent", "fee_amount", "enable", "notify_url"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("payment method list item missing %q: %s", key, payload)
		}
	}
}

func TestCouponListItemKeepsListPayloadLean(t *testing.T) {
	payload, err := json.Marshal(CouponListItem{
		Id:         1,
		Name:       "Launch",
		Code:       "LAUNCH",
		Count:      100,
		Type:       1,
		Discount:   20,
		StartTime:  1700000000000,
		ExpireTime: 1700086400000,
		UserLimit:  1,
		UsedCount:  7,
		Enable:     true,
	})
	if err != nil {
		t.Fatalf("json.Marshal(CouponListItem) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(CouponListItem) error = %v", err)
	}
	for _, key := range []string{"subscribe", "created_at", "updated_at"} {
		if _, ok := got[key]; ok {
			t.Fatalf("coupon list item must not expose %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "name", "code", "count", "type", "discount", "start_time", "expire_time", "user_limit", "used_count", "enable"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("coupon list item missing %q: %s", key, payload)
		}
	}
}

func TestXrayTemplateListAndCatalogPayloadShapes(t *testing.T) {
	listPayload, err := json.Marshal(XrayTemplateListItem{
		Id:          1,
		Name:        "Inbound",
		Type:        "inbound",
		Description: "public list row",
		Enabled:     true,
		UpdatedAt:   1700000000000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(XrayTemplateListItem) error = %v", err)
	}

	var list map[string]any
	if err := json.Unmarshal(listPayload, &list); err != nil {
		t.Fatalf("json.Unmarshal(XrayTemplateListItem) error = %v", err)
	}
	for _, key := range []string{"config", "config_template", "variables_schema", "default_variables", "subscription_meta", "created_at"} {
		if _, ok := list[key]; ok {
			t.Fatalf("xray template list item must not expose %q: %s", key, listPayload)
		}
	}
	for _, key := range []string{"id", "name", "type", "description", "enabled", "updated_at"} {
		if _, ok := list[key]; !ok {
			t.Fatalf("xray template list item missing %q: %s", key, listPayload)
		}
	}

	catalogPayload, err := json.Marshal(XrayTemplateCatalogItem{
		Id:               1,
		Name:             "Inbound",
		Type:             "inbound",
		Config:           json.RawMessage(`{}`),
		ConfigTemplate:   "{}",
		VariablesSchema:  json.RawMessage(`{}`),
		DefaultVariables: json.RawMessage(`{}`),
		SubscriptionMeta: json.RawMessage(`{}`),
		Description:      "catalog row",
		Enabled:          true,
	})
	if err != nil {
		t.Fatalf("json.Marshal(XrayTemplateCatalogItem) error = %v", err)
	}

	var catalog map[string]any
	if err := json.Unmarshal(catalogPayload, &catalog); err != nil {
		t.Fatalf("json.Unmarshal(XrayTemplateCatalogItem) error = %v", err)
	}
	for _, key := range []string{"created_at", "updated_at"} {
		if _, ok := catalog[key]; ok {
			t.Fatalf("xray template catalog item must not expose %q: %s", key, catalogPayload)
		}
	}
	for _, key := range []string{"id", "name", "type", "config", "config_template", "variables_schema", "default_variables", "subscription_meta", "description", "enabled"} {
		if _, ok := catalog[key]; !ok {
			t.Fatalf("xray template catalog item missing %q: %s", key, catalogPayload)
		}
	}
}

func TestContentListPayloadShapes(t *testing.T) {
	documentPayload, err := json.Marshal(DocumentListItem{
		Id:        1,
		Title:     "Admin document",
		Tags:      []string{"guide"},
		Show:      true,
		CreatedAt: 1700000000000,
		UpdatedAt: 1700000005000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(DocumentListItem) error = %v", err)
	}
	var document map[string]any
	if err := json.Unmarshal(documentPayload, &document); err != nil {
		t.Fatalf("json.Unmarshal(DocumentListItem) error = %v", err)
	}
	for _, key := range []string{"content"} {
		if _, ok := document[key]; ok {
			t.Fatalf("document list item must not expose %q: %s", key, documentPayload)
		}
	}
	for _, key := range []string{"id", "title", "tags", "show", "created_at", "updated_at"} {
		if _, ok := document[key]; !ok {
			t.Fatalf("document list item missing %q: %s", key, documentPayload)
		}
	}

	publicDocumentPayload, err := json.Marshal(PublicDocumentListItem{
		Id:        1,
		Title:     "Public document",
		Tags:      []string{"guide"},
		UpdatedAt: 1700000005000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(PublicDocumentListItem) error = %v", err)
	}
	var publicDocument map[string]any
	if err := json.Unmarshal(publicDocumentPayload, &publicDocument); err != nil {
		t.Fatalf("json.Unmarshal(PublicDocumentListItem) error = %v", err)
	}
	for _, key := range []string{"content", "show", "created_at"} {
		if _, ok := publicDocument[key]; ok {
			t.Fatalf("public document list item must not expose %q: %s", key, publicDocumentPayload)
		}
	}
	for _, key := range []string{"id", "title", "tags", "updated_at"} {
		if _, ok := publicDocument[key]; !ok {
			t.Fatalf("public document list item missing %q: %s", key, publicDocumentPayload)
		}
	}

	announcementPayload, err := json.Marshal(AnnouncementListItem{
		Id:        1,
		Title:     "Maintenance",
		Show:      true,
		Pinned:    false,
		Popup:     true,
		CreatedAt: 1700000000000,
		UpdatedAt: 1700000005000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(AnnouncementListItem) error = %v", err)
	}
	var announcement map[string]any
	if err := json.Unmarshal(announcementPayload, &announcement); err != nil {
		t.Fatalf("json.Unmarshal(AnnouncementListItem) error = %v", err)
	}
	if _, ok := announcement["content"]; ok {
		t.Fatalf("announcement list item must not expose content: %s", announcementPayload)
	}

	adsPayload, err := json.Marshal(AdsListItem{
		Id:          1,
		Title:       "Banner",
		Type:        "home",
		Description: "desc",
		TargetURL:   "https://example.test",
		StartTime:   1700000000000,
		EndTime:     1700000005000,
		Status:      1,
		CreatedAt:   1700000000000,
		UpdatedAt:   1700000005000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(AdsListItem) error = %v", err)
	}
	var ads map[string]any
	if err := json.Unmarshal(adsPayload, &ads); err != nil {
		t.Fatalf("json.Unmarshal(AdsListItem) error = %v", err)
	}
	if _, ok := ads["content"]; ok {
		t.Fatalf("ads list item must not expose content: %s", adsPayload)
	}
}

func TestUserListPayloadShape(t *testing.T) {
	payload, err := json.Marshal(UserListItem{
		Id:              1,
		Balance:         100,
		Commission:      20,
		GiftAmount:      30,
		MemberExpiredAt: 1700000000000,
		IsMember:        true,
		ReferCode:       "REF",
		RefererId:       2,
		Enable:          true,
		PrimaryAuthMethod: UserAuthMethod{
			AuthType:       "email",
			AuthIdentifier: "alpha@example.test",
			Verified:       true,
		},
		CreatedAt: 1700000000000,
		DeletedAt: 1700000005000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(UserListItem) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(UserListItem) error = %v", err)
	}
	for _, key := range []string{
		"avatar",
		"referral_percentage",
		"only_first_purchase",
		"telegram",
		"is_admin",
		"enable_balance_notify",
		"enable_login_notify",
		"enable_subscribe_notify",
		"enable_trade_notify",
		"auth_methods",
		"rules",
		"updated_at",
	} {
		if _, ok := got[key]; ok {
			t.Fatalf("user list item must not expose %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "balance", "commission", "gift_amount", "member_expired_at", "is_member", "refer_code", "referer_id", "enable", "primary_auth_method", "created_at", "deleted_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("user list item missing %q: %s", key, payload)
		}
	}
}

func TestUserKeepsBooleanAndArrayResponseShape(t *testing.T) {
	payload, err := json.Marshal(User{
		Id:          1,
		IsAdmin:     false,
		AuthMethods: []UserAuthMethod{},
		Rules:       []string{},
	})
	if err != nil {
		t.Fatalf("json.Marshal(User) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(User) error = %v", err)
	}
	isAdmin, ok := got["is_admin"].(bool)
	if !ok || isAdmin {
		t.Fatalf("user response must include stable false is_admin bool: %s", payload)
	}
	for _, key := range []string{"auth_methods", "rules"} {
		list, ok := got[key].([]any)
		if !ok || len(list) != 0 {
			t.Fatalf("user response must include stable empty %s array: %s", key, payload)
		}
	}
	if _, ok := got["user_devices"]; ok {
		t.Fatalf("user response must not inline devices; use device list endpoints instead: %s", payload)
	}
}

func TestUserSubscribeInfoKeepsScopePayloadLean(t *testing.T) {
	payload, err := json.Marshal(UserSubscribeInfo{
		Id:       1,
		Nodes:    []UserSubscribeNodeInfo{},
		Profiles: []UserSubscribeProfile{},
		Presets:  []UserSubscribePreset{},
	})
	if err != nil {
		t.Fatalf("json.Marshal(UserSubscribeInfo) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(UserSubscribeInfo) error = %v", err)
	}
	for _, key := range []string{
		"user_id",
		"order_id",
		"subscribe_id",
		"product_name",
		"start_time",
		"expire_time",
		"finished_at",
		"reset_time",
		"traffic",
		"download",
		"upload",
		"token",
		"status",
		"created_at",
		"updated_at",
		"is_try_out",
	} {
		if _, ok := got[key]; ok {
			t.Fatalf("user subscribe scope payload must not expose redundant %q: %s", key, payload)
		}
	}
	for _, key := range []string{"nodes", "profiles", "presets"} {
		list, ok := got[key].([]any)
		if !ok || len(list) != 0 {
			t.Fatalf("user subscribe scope payload must include stable empty %s array: %s", key, payload)
		}
	}
}

func TestPublicUserSubscribeOmitsInternalListFields(t *testing.T) {
	payload, err := json.Marshal(PublicUserSubscribe{
		Id:          1,
		Subscribe:   PublicUserSubscribeProduct{Name: "Plan", Discount: []SubscribeDiscount{}},
		ProductName: "Plan",
		ExpireTime:  1700000000000,
		ResetTime:   1701000000000,
		Traffic:     1024,
		Download:    256,
		Upload:      128,
		Token:       "subscribe-token",
		Status:      1,
		Short:       "short",
	})
	if err != nil {
		t.Fatalf("json.Marshal(PublicUserSubscribe) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(PublicUserSubscribe) error = %v", err)
	}
	for _, key := range []string{
		"user_id",
		"order_id",
		"subscribe_id",
		"start_time",
		"finished_at",
		"created_at",
		"updated_at",
	} {
		if _, ok := got[key]; ok {
			t.Fatalf("public user subscribe must not expose redundant %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "subscribe", "product_name", "expire_time", "reset_time", "traffic", "download", "upload", "token", "status", "short"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("public user subscribe missing %q: %s", key, payload)
		}
	}
	subscribe, ok := got["subscribe"].(map[string]any)
	if !ok {
		t.Fatalf("public user subscribe product must be an object: %s", payload)
	}
	for _, key := range []string{
		"language",
		"description",
		"inventory",
		"quota",
		"nodes",
		"node_tags",
		"show",
		"user_visible",
		"sell",
		"sort",
		"deduction_ratio",
		"reset_cycle",
		"renewal_reset",
		"created_at",
		"updated_at",
	} {
		if _, ok := subscribe[key]; ok {
			t.Fatalf("public user subscribe product must not expose redundant %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "name", "unit_price", "unit_time", "discount", "replacement", "traffic", "speed_limit", "device_limit", "allow_renewal", "allow_reset_traffic", "allow_deduction", "show_original_price"} {
		if _, ok := subscribe[key]; !ok {
			t.Fatalf("public user subscribe product missing %q: %s", key, payload)
		}
	}
	if discount, ok := subscribe["discount"].([]any); !ok || len(discount) != 0 {
		t.Fatalf("public user subscribe product must keep empty discount as []: %s", payload)
	}
}

func TestUserSubscribeListItemOmitsAccessAndFullProductFields(t *testing.T) {
	payload, err := json.Marshal(UserSubscribeListItem{
		Id:          1,
		UserId:      2,
		SubscribeId: 3,
		Subscribe: UserSubscribeProductSummary{
			Id:          3,
			Name:        "Plan",
			SpeedLimit:  1048576,
			DeviceLimit: 3,
		},
		ProductName: "Plan",
		ExpireTime:  1700000000000,
		ResetTime:   0,
		Traffic:     1024,
		Download:    256,
		Upload:      128,
		Status:      1,
		CreatedAt:   1690000000000,
		UpdatedAt:   1690000000000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(UserSubscribeListItem) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(UserSubscribeListItem) error = %v", err)
	}
	for _, key := range []string{"order_id", "finished_at", "token", "short"} {
		if _, ok := got[key]; ok {
			t.Fatalf("admin user subscribe list item must not expose %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "user_id", "subscribe_id", "subscribe", "product_name", "start_time", "expire_time", "reset_time", "traffic", "download", "upload", "status", "created_at", "updated_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("admin user subscribe list item missing %q: %s", key, payload)
		}
	}
	subscribe, ok := got["subscribe"].(map[string]any)
	if !ok {
		t.Fatalf("admin user subscribe list item missing subscribe summary: %s", payload)
	}
	for _, key := range []string{"id", "name", "speed_limit", "device_limit"} {
		if _, ok := subscribe[key]; !ok {
			t.Fatalf("admin user subscribe product summary missing %q: %s", key, payload)
		}
	}
	for _, key := range []string{"unit_price", "unit_time", "discount", "replacement", "traffic", "nodes", "node_tags", "allow_deduction", "allow_renewal", "allow_reset_traffic", "reset_cycle", "show_original_price"} {
		if _, ok := subscribe[key]; ok {
			t.Fatalf("admin user subscribe product summary must not expose %q: %s", key, payload)
		}
	}
}

func TestPublicSubscribeKeepsProductListPayloadLean(t *testing.T) {
	payload, err := json.Marshal(PublicSubscribe{
		Id:                1,
		Name:              "Plan",
		Description:       "public plan",
		UnitPrice:         1000,
		UnitTime:          "month",
		Discount:          []SubscribeDiscount{},
		Replacement:       100,
		Traffic:           1024,
		SpeedLimit:        100,
		DeviceLimit:       3,
		UserVisible:       true,
		Sell:              true,
		ShowOriginalPrice: true,
	})
	if err != nil {
		t.Fatalf("json.Marshal(PublicSubscribe) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(PublicSubscribe) error = %v", err)
	}
	for _, key := range []string{
		"language",
		"inventory",
		"quota",
		"nodes",
		"node_tags",
		"show",
		"sort",
		"deduction_ratio",
		"allow_deduction",
		"allow_renewal",
		"allow_reset_traffic",
		"reset_cycle",
		"renewal_reset",
		"created_at",
		"updated_at",
	} {
		if _, ok := got[key]; ok {
			t.Fatalf("public subscribe must not expose redundant %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "name", "description", "unit_price", "unit_time", "discount", "replacement", "traffic", "speed_limit", "device_limit", "user_visible", "sell", "show_original_price"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("public subscribe missing %q: %s", key, payload)
		}
	}
	if discount, ok := got["discount"].([]any); !ok || len(discount) != 0 {
		t.Fatalf("public subscribe must keep empty discount as []: %s", payload)
	}
}

func TestAdminSubscribeItemKeepsListPayloadLean(t *testing.T) {
	payload, err := json.Marshal(SubscribeItem{
		Id:           1,
		Name:         "Pro",
		Language:     "en",
		UnitPrice:    1000,
		UnitTime:     "month",
		Replacement:  1200,
		Inventory:    100,
		Traffic:      1024,
		SpeedLimit:   200,
		DeviceLimit:  3,
		Quota:        10,
		Show:         true,
		UserVisible:  true,
		Sell:         true,
		AllowRenewal: true,
		ResetCycle:   2,
		Sort:         5,
		CreatedAt:    1700000000000,
		UpdatedAt:    1700000005000,
		Sold:         12,
	})
	if err != nil {
		t.Fatalf("json.Marshal(SubscribeItem) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(SubscribeItem) error = %v", err)
	}
	for _, key := range []string{
		"description",
		"discount",
		"nodes",
		"node_tags",
		"deduction_ratio",
		"allow_deduction",
		"allow_reset_traffic",
		"renewal_reset",
		"show_original_price",
	} {
		if _, ok := got[key]; ok {
			t.Fatalf("admin subscribe list item must not expose %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "name", "language", "unit_price", "unit_time", "replacement", "inventory", "traffic", "speed_limit", "device_limit", "quota", "show", "user_visible", "sell", "allow_renewal", "reset_cycle", "sort", "created_at", "updated_at", "sold"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("admin subscribe list item missing %q: %s", key, payload)
		}
	}
}

func TestTicketListItemsKeepLeanResponseShapes(t *testing.T) {
	adminPayload, err := json.Marshal(TicketListItem{
		Id:        1,
		Title:     "Need help",
		UserId:    7,
		Status:    1,
		UpdatedAt: 1700000000000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(TicketListItem) error = %v", err)
	}

	var admin map[string]any
	if err := json.Unmarshal(adminPayload, &admin); err != nil {
		t.Fatalf("json.Unmarshal(TicketListItem) error = %v", err)
	}
	for _, key := range []string{"description", "follow", "created_at"} {
		if _, ok := admin[key]; ok {
			t.Fatalf("admin ticket list item must not expose redundant %q: %s", key, adminPayload)
		}
	}
	for _, key := range []string{"id", "title", "user_id", "status", "updated_at"} {
		if _, ok := admin[key]; !ok {
			t.Fatalf("admin ticket list item missing %q: %s", key, adminPayload)
		}
	}

	userPayload, err := json.Marshal(UserTicketListItem{
		Id:          1,
		Title:       "Need help",
		Description: "bounded preview",
		Status:      1,
		UpdatedAt:   1700000000000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(UserTicketListItem) error = %v", err)
	}

	var user map[string]any
	if err := json.Unmarshal(userPayload, &user); err != nil {
		t.Fatalf("json.Unmarshal(UserTicketListItem) error = %v", err)
	}
	for _, key := range []string{"user_id", "follow", "created_at"} {
		if _, ok := user[key]; ok {
			t.Fatalf("user ticket list item must not expose redundant %q: %s", key, userPayload)
		}
	}
	for _, key := range []string{"id", "title", "description", "status", "updated_at"} {
		if _, ok := user[key]; !ok {
			t.Fatalf("user ticket list item missing %q: %s", key, userPayload)
		}
	}
}

func TestPublicOrderListItemKeepsLeanResponseShape(t *testing.T) {
	payload, err := json.Marshal(PublicOrder{
		Id:             1,
		OrderNo:        "alpha-order",
		Type:           1,
		Quantity:       1,
		Price:          1000,
		Amount:         900,
		GiftAmount:     100,
		Discount:       100,
		CouponDiscount: 50,
		FeeAmount:      5,
		Status:         2,
		SubscribeId:    3,
		ProductName:    "Plan",
		CreatedAt:      1700000000000,
		UpdatedAt:      1700000000000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(PublicOrder) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(PublicOrder) error = %v", err)
	}
	for _, key := range []string{
		"user_id",
		"coupon",
		"commission",
		"payment",
		"method",
		"trade_no",
		"subscribe",
	} {
		if _, ok := got[key]; ok {
			t.Fatalf("public order list item must not expose redundant %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "order_no", "type", "quantity", "price", "amount", "gift_amount", "discount", "coupon_discount", "fee_amount", "status", "subscribe_id", "product_name", "created_at", "updated_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("public order list item missing %q: %s", key, payload)
		}
	}
}

func TestAdminOrderListItemKeepsLeanResponseShape(t *testing.T) {
	payload, err := json.Marshal(OrderListItem{
		Id:             1,
		UserId:         7,
		OrderNo:        "alpha-order",
		Type:           1,
		Quantity:       1,
		Price:          1000,
		Amount:         900,
		GiftAmount:     100,
		Discount:       100,
		CouponDiscount: 50,
		Payment: PaymentMethodSummary{
			Id:       2,
			Name:     "Stripe",
			Platform: "stripe",
		},
		FeeAmount:   5,
		TradeNo:     "trade-no",
		Status:      2,
		SubscribeId: 3,
		ProductName: "Plan",
		UpdatedAt:   1700000000000,
	})
	if err != nil {
		t.Fatalf("json.Marshal(OrderListItem) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(OrderListItem) error = %v", err)
	}
	for _, key := range []string{
		"coupon",
		"commission",
		"method",
		"subscribe",
		"membership_plan",
		"created_at",
	} {
		if _, ok := got[key]; ok {
			t.Fatalf("admin order list item must not expose redundant %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "user_id", "order_no", "type", "quantity", "price", "amount", "gift_amount", "discount", "coupon_discount", "payment", "fee_amount", "trade_no", "status", "subscribe_id", "product_name", "updated_at"} {
		if _, ok := got[key]; !ok {
			t.Fatalf("admin order list item missing %q: %s", key, payload)
		}
	}
	payment, ok := got["payment"].(map[string]any)
	if !ok {
		t.Fatalf("admin order payment summary must be an object: %s", payload)
	}
	for _, key := range []string{"id", "name", "platform"} {
		if _, ok := payment[key]; !ok {
			t.Fatalf("admin order payment summary missing %q: %s", key, payload)
		}
	}
	for _, key := range []string{"description", "icon", "fee_mode", "fee_percent", "fee_amount"} {
		if _, ok := payment[key]; ok {
			t.Fatalf("admin order payment summary must not expose %q: %s", key, payload)
		}
	}
}

func TestOrderDetailSubscribeKeepsPaymentPageProductShape(t *testing.T) {
	payload, err := json.Marshal(OrderDetail{
		Id:      1,
		OrderNo: "alpha-order",
		Type:    1,
		Subscribe: OrderSubscribeProduct{
			Id:                3,
			Name:              "Plan",
			UnitPrice:         1000,
			UnitTime:          "month",
			Traffic:           1024,
			SpeedLimit:        100,
			DeviceLimit:       3,
			ShowOriginalPrice: true,
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(OrderDetail) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(OrderDetail) error = %v", err)
	}
	subscribe, ok := got["subscribe"].(map[string]any)
	if !ok {
		t.Fatalf("order detail subscribe must be an object: %s", payload)
	}
	for _, key := range []string{
		"language",
		"description",
		"discount",
		"replacement",
		"inventory",
		"quota",
		"nodes",
		"node_tags",
		"show",
		"user_visible",
		"sell",
		"sort",
		"deduction_ratio",
		"allow_deduction",
		"allow_renewal",
		"allow_reset_traffic",
		"reset_cycle",
		"renewal_reset",
		"created_at",
		"updated_at",
	} {
		if _, ok := subscribe[key]; ok {
			t.Fatalf("order detail subscribe must not expose redundant %q: %s", key, payload)
		}
	}
	for _, key := range []string{"id", "name", "unit_price", "unit_time", "traffic", "speed_limit", "device_limit", "show_original_price"} {
		if _, ok := subscribe[key]; !ok {
			t.Fatalf("order detail subscribe missing %q: %s", key, payload)
		}
	}
}

func TestQueryPurchaseOrderResponseSubscribeKeepsPaymentPageProductShape(t *testing.T) {
	payload, err := json.Marshal(QueryPurchaseOrderResponse{
		OrderNo: "alpha-order",
		Subscribe: OrderSubscribeProduct{
			Id:        3,
			Name:      "Plan",
			UnitPrice: 1000,
			UnitTime:  "month",
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(QueryPurchaseOrderResponse) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(QueryPurchaseOrderResponse) error = %v", err)
	}
	subscribe, ok := got["subscribe"].(map[string]any)
	if !ok {
		t.Fatalf("purchase order subscribe must be an object: %s", payload)
	}
	if _, ok := subscribe["nodes"]; ok {
		t.Fatalf("purchase order subscribe must not expose node topology: %s", payload)
	}
	if _, ok := subscribe["discount"]; ok {
		t.Fatalf("purchase order subscribe must not expose product discount rules: %s", payload)
	}
	if _, ok := subscribe["unit_price"]; !ok {
		t.Fatalf("purchase order subscribe missing unit_price: %s", payload)
	}
}

func TestServerForwardTableUsesCanonicalItemsJSONKey(t *testing.T) {
	payload, err := json.Marshal(QueryServerForwardTableResponse{
		IP: "192.0.2.10",
		Items: []ServerForwardItem{
			{
				LocalPort:  8443,
				TargetHost: "198.51.100.20",
				TargetPort: 443,
			},
		},
	})
	if err != nil {
		t.Fatalf("json.Marshal(QueryServerForwardTableResponse) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(QueryServerForwardTableResponse) error = %v", err)
	}
	if _, ok := got["forwarded_items"]; !ok {
		t.Fatalf("forward table response must use forwarded_items JSON key: %s", payload)
	}
	if _, ok := got["forwared_items"]; ok {
		t.Fatalf("forward table response must not use misspelled forwared_items key: %s", payload)
	}
	if warnings, ok := got["warnings"].([]any); !ok || len(warnings) != 0 {
		t.Fatalf("forward table response must include stable empty warnings array: %s", payload)
	}
}

func TestServerXrayUserInboundsKeyIsStable(t *testing.T) {
	payload, err := json.Marshal(ServerXrayUser{
		UserSubscribeID: 10,
		UserID:          1,
		Email:           "u_10",
		UUID:            "uuid-10",
		Password:        "uuid-10",
		Inbounds:        []string{"vless-in"},
		Enabled:         true,
	})
	if err != nil {
		t.Fatalf("json.Marshal(ServerXrayUser) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(ServerXrayUser) error = %v", err)
	}
	inbounds, ok := got["inbounds"].([]any)
	if !ok || len(inbounds) != 1 || inbounds[0] != "vless-in" {
		t.Fatalf("server xray user must include inbounds array: %s", payload)
	}
}

func TestPreviewServerXrayConfigResponseKeepsPreviewCollections(t *testing.T) {
	payload, err := json.Marshal(PreviewServerXrayConfigResponse{
		QueryServerXrayConfigResponse: QueryServerXrayConfigResponse{
			Inbounds:  []json.RawMessage{},
			Outbounds: []json.RawMessage{},
			Templates: []XrayTemplate{},
		},
		SubscriptionProxies: []json.RawMessage{},
		Variables:           map[string]XrayTemplateVariablePreview{},
	})
	if err != nil {
		t.Fatalf("json.Marshal(PreviewServerXrayConfigResponse) error = %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(payload, &got); err != nil {
		t.Fatalf("json.Unmarshal(PreviewServerXrayConfigResponse) error = %v", err)
	}
	for _, key := range []string{"inbounds", "outbounds", "templates", "subscription_proxies"} {
		list, ok := got[key].([]any)
		if !ok || len(list) != 0 {
			t.Fatalf("preview xray config response must include stable empty %s array: %s", key, payload)
		}
	}
	variables, ok := got["variables"].(map[string]any)
	if !ok || len(variables) != 0 {
		t.Fatalf("preview xray config response must include stable empty variables object: %s", payload)
	}
}

func TestStatisticsResponsesDoNotExposeNestedLists(t *testing.T) {
	revenuePayload, err := json.Marshal(RevenueStatisticsResponse{
		Period:      "7d",
		Granularity: "day",
		Series:      []OrdersStatistics{},
		Today:       OrdersStatistics{},
		Monthly:     OrdersStatistics{},
		All:         OrdersStatistics{},
	})
	if err != nil {
		t.Fatalf("json.Marshal(RevenueStatisticsResponse) error = %v", err)
	}
	var revenue map[string]any
	if err := json.Unmarshal(revenuePayload, &revenue); err != nil {
		t.Fatalf("json.Unmarshal(RevenueStatisticsResponse) error = %v", err)
	}
	series, ok := revenue["series"].([]any)
	if !ok || len(series) != 0 {
		t.Fatalf("revenue statistics response must include stable empty series array: %s", revenuePayload)
	}
	for _, key := range []string{"today", "monthly", "all"} {
		point, ok := revenue[key].(map[string]any)
		if !ok {
			t.Fatalf("revenue statistics %s must be an object: %s", key, revenuePayload)
		}
		if _, ok := point["list"]; ok {
			t.Fatalf("revenue statistics %s must not expose nested list: %s", key, revenuePayload)
		}
	}

	userPayload, err := json.Marshal(UserStatisticsResponse{
		Period:      "7d",
		Granularity: "day",
		Series:      []UserStatistics{},
		Today:       UserStatistics{},
		Monthly:     UserStatistics{},
		All:         UserStatistics{},
	})
	if err != nil {
		t.Fatalf("json.Marshal(UserStatisticsResponse) error = %v", err)
	}
	var users map[string]any
	if err := json.Unmarshal(userPayload, &users); err != nil {
		t.Fatalf("json.Unmarshal(UserStatisticsResponse) error = %v", err)
	}
	series, ok = users["series"].([]any)
	if !ok || len(series) != 0 {
		t.Fatalf("user statistics response must include stable empty series array: %s", userPayload)
	}
	for _, key := range []string{"today", "monthly", "all"} {
		point, ok := users[key].(map[string]any)
		if !ok {
			t.Fatalf("user statistics %s must be an object: %s", key, userPayload)
		}
		if _, ok := point["list"]; ok {
			t.Fatalf("user statistics %s must not expose nested list: %s", key, userPayload)
		}
	}
}

func apiTypeBody(t *testing.T, source, typeName string) string {
	t.Helper()
	re := regexp.MustCompile(`(?s)\b` + regexp.QuoteMeta(typeName) + `\s*\{(.*?)\n\s*\}`)
	match := re.FindStringSubmatch(source)
	if len(match) != 2 {
		t.Fatalf("%s type body not found in api source", typeName)
	}
	return match[1]
}

func apiJSONFieldSet(body string) map[string]struct{} {
	re := regexp.MustCompile("`[^`]*json:\"([^\"]+)\"")
	fields := make(map[string]struct{})
	for _, match := range re.FindAllStringSubmatch(body, -1) {
		if len(match) != 2 {
			continue
		}
		name := strings.Split(match[1], ",")[0]
		if name != "" && name != "-" {
			fields[name] = struct{}{}
		}
	}
	return fields
}

func structJSONFieldSet(model reflect.Type) map[string]struct{} {
	fields := make(map[string]struct{})
	for i := 0; i < model.NumField(); i++ {
		tag := model.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name != "" && name != "-" {
			fields[name] = struct{}{}
		}
	}
	return fields
}

type contractStruct struct {
	Name string
	File string
	Body string
}

func parseAPIContractStructs(t *testing.T, root string) ([]contractStruct, map[string]string) {
	t.Helper()
	var structs []contractStruct
	bodies := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || !strings.HasSuffix(path, ".api") {
			return nil
		}
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, item := range parseAPIContractStructsFromSource(path, string(source)) {
			structs = append(structs, item)
			if existing, ok := bodies[item.Name]; ok && strings.TrimSpace(existing) != strings.TrimSpace(item.Body) {
				t.Fatalf("%s has conflicting .api definitions; latest conflict in %s", item.Name, item.File)
			}
			bodies[item.Name] = item.Body
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk api sources: %v", err)
	}
	return structs, bodies
}

func parseAPIContractStructsFromSource(file, source string) []contractStruct {
	matches := apiContractStructRE.FindAllStringSubmatchIndex(source, -1)
	structs := make([]contractStruct, 0, len(matches))
	for _, match := range matches {
		name := source[match[2]:match[3]]
		bodyStart := match[1]
		bodyEnd := matchingContractBraceEnd(source, bodyStart-1)
		if bodyEnd < 0 {
			continue
		}
		structs = append(structs, contractStruct{
			Name: name,
			File: file,
			Body: source[bodyStart:bodyEnd],
		})
	}
	return structs
}

func parseGeneratedTypeStructs(t *testing.T, file string) map[string]string {
	t.Helper()
	source, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read generated types: %v", err)
	}
	text := string(source)
	matches := generatedContractStructRE.FindAllStringSubmatchIndex(text, -1)
	aliases := generatedContractAliasRE.FindAllStringSubmatch(text, -1)
	bodies := make(map[string]string, len(matches)+len(aliases))
	for _, match := range matches {
		name := text[match[2]:match[3]]
		bodyStart := match[1]
		bodyEnd := matchingContractBraceEnd(text, bodyStart-1)
		if bodyEnd < 0 {
			continue
		}
		bodies[name] = text[bodyStart:bodyEnd]
	}
	for _, alias := range aliases {
		if len(alias) != 3 {
			continue
		}
		if body, ok := bodies[alias[2]]; ok {
			bodies[alias[1]] = body
		}
	}
	return bodies
}

func matchingContractBraceEnd(source string, openBrace int) int {
	depth := 0
	for i := openBrace; i < len(source); i++ {
		switch source[i] {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return i
			}
		}
	}
	return -1
}

func contractTaggedFieldSet(typeName string, bodies map[string]string, seen map[string]bool) map[string]struct{} {
	if seen == nil {
		seen = make(map[string]bool)
	}
	if seen[typeName] {
		return map[string]struct{}{}
	}
	body, ok := bodies[typeName]
	if !ok {
		return map[string]struct{}{}
	}
	seen[typeName] = true

	fields := make(map[string]struct{})
	for _, raw := range strings.Split(body, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		taggedFields := contractTaggedFields(line)
		for field := range taggedFields {
			fields[field] = struct{}{}
		}
		if len(taggedFields) > 0 {
			continue
		}
		if embedded := contractEmbeddedType(line); embedded != "" {
			for field := range contractTaggedFieldSet(embedded, bodies, seen) {
				fields[field] = struct{}{}
			}
		}
	}
	return fields
}

func contractTaggedFields(line string) map[string]struct{} {
	fields := make(map[string]struct{})
	tagStart := strings.IndexByte(line, '`')
	if tagStart < 0 {
		return fields
	}
	tagEnd := strings.LastIndexByte(line, '`')
	if tagEnd <= tagStart {
		return fields
	}
	tag := line[tagStart+1 : tagEnd]
	for _, match := range contractTagRE.FindAllStringSubmatch(tag, -1) {
		name := strings.Split(match[2], ",")[0]
		if name != "" && name != "-" {
			fields[match[1]+":"+match[2]] = struct{}{}
		}
	}
	return fields
}

func contractEmbeddedType(line string) string {
	line = strings.Split(line, "//")[0]
	line = strings.TrimSpace(line)
	if line == "" || strings.ContainsAny(line, " \t`") {
		return ""
	}
	if !contractEmbeddedTypeRE.MatchString(line) {
		return ""
	}
	return line
}

func sortedFieldSet(fields map[string]struct{}) []string {
	resp := make([]string, 0, len(fields))
	for field := range fields {
		resp = append(resp, field)
	}
	sort.Strings(resp)
	return resp
}

func TestSubscribeApplicationResponsesKeepDownloadLinkObject(t *testing.T) {
	applicationPayload, err := json.Marshal(SubscribeApplication{
		Id:           1,
		Name:         "Alpha",
		Enabled:      true,
		IsDefault:    true,
		OutputFormat: "uri",
		DownloadLink: DownloadLink{},
	})
	if err != nil {
		t.Fatalf("json.Marshal(SubscribeApplication) error = %v", err)
	}
	var application map[string]any
	if err := json.Unmarshal(applicationPayload, &application); err != nil {
		t.Fatalf("json.Unmarshal(SubscribeApplication) error = %v", err)
	}
	link, ok := application["download_link"].(map[string]any)
	if !ok || len(link) != 0 {
		t.Fatalf("subscribe application response must include stable empty download_link object: %s", applicationPayload)
	}

	listPayload, err := json.Marshal(SubscribeApplicationListItem{
		Id:           1,
		Name:         "Alpha",
		Enabled:      true,
		IsDefault:    true,
		OutputFormat: "uri",
		DownloadLink: DownloadLink{},
	})
	if err != nil {
		t.Fatalf("json.Marshal(SubscribeApplicationListItem) error = %v", err)
	}
	var listItem map[string]any
	if err := json.Unmarshal(listPayload, &listItem); err != nil {
		t.Fatalf("json.Unmarshal(SubscribeApplicationListItem) error = %v", err)
	}
	if _, ok := listItem["template"]; ok {
		t.Fatalf("subscribe application list item must not expose template payload: %s", listPayload)
	}
	link, ok = listItem["download_link"].(map[string]any)
	if !ok || len(link) != 0 {
		t.Fatalf("subscribe application list item must include stable empty download_link object: %s", listPayload)
	}

	clientPayload, err := json.Marshal(SubscribeClient{
		Id:           1,
		Name:         "Alpha",
		Enabled:      true,
		IsDefault:    true,
		DownloadLink: DownloadLink{},
	})
	if err != nil {
		t.Fatalf("json.Marshal(SubscribeClient) error = %v", err)
	}
	var client map[string]any
	if err := json.Unmarshal(clientPayload, &client); err != nil {
		t.Fatalf("json.Unmarshal(SubscribeClient) error = %v", err)
	}
	link, ok = client["download_link"].(map[string]any)
	if !ok || len(link) != 0 {
		t.Fatalf("subscribe client response must include stable empty download_link object: %s", clientPayload)
	}
}

func TestQueryServerXrayUsersResponseOmitsUsersOnlyForDelta(t *testing.T) {
	deltaPayload, err := json.Marshal(QueryServerXrayUsersResponse{
		Version: "v1",
		Full:    false,
		Users:   nil,
	})
	if err != nil {
		t.Fatalf("json.Marshal(delta QueryServerXrayUsersResponse) error = %v", err)
	}
	var delta map[string]any
	if err := json.Unmarshal(deltaPayload, &delta); err != nil {
		t.Fatalf("json.Unmarshal(delta QueryServerXrayUsersResponse) error = %v", err)
	}
	if _, ok := delta["users"]; ok {
		t.Fatalf("delta xray users response should omit unchanged users payload: %s", deltaPayload)
	}

	fullPayload, err := json.Marshal(QueryServerXrayUsersResponse{
		Version: "v1",
		Full:    true,
		Users:   nil,
	})
	if err != nil {
		t.Fatalf("json.Marshal(full QueryServerXrayUsersResponse) error = %v", err)
	}
	var full map[string]any
	if err := json.Unmarshal(fullPayload, &full); err != nil {
		t.Fatalf("json.Unmarshal(full QueryServerXrayUsersResponse) error = %v", err)
	}
	users, ok := full["users"].([]any)
	if !ok || len(users) != 0 {
		t.Fatalf("full xray users response should include stable empty users array: %s", fullPayload)
	}
}
