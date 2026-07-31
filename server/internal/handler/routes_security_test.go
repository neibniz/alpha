package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/neibniz/alpha/server/internal/svc"
	"github.com/neibniz/alpha/server/pkg/xerr"
	"github.com/gin-gonic/gin"
)

func TestPortalOrderStatusRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHandlers(router, &svc.ServiceContext{})

	request := httptest.NewRequest(http.MethodGet, "/v1/public/portal/order/status?order_no=alpha-order", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Code uint32 `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != xerr.ErrorTokenEmpty {
		t.Fatalf("response code = %d, want %d", response.Code, xerr.ErrorTokenEmpty)
	}
}

func TestAdminOrderDetailRequiresAuthentication(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	RegisterHandlers(router, &svc.ServiceContext{})

	request := httptest.NewRequest(http.MethodGet, "/v1/admin/order/detail?id=1", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusOK)
	}
	var response struct {
		Code uint32 `json:"code"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Code != xerr.ErrorTokenEmpty {
		t.Fatalf("response code = %d, want %d", response.Code, xerr.ErrorTokenEmpty)
	}
}
