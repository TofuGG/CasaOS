package route

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/IceWhaleTech/CasaOS/model"
	"github.com/IceWhaleTech/CasaOS/service"
)

// ---- stubs ---------------------------------------------------------------

// stubRepo satisfies service.Repository. Only System() is used by the routes
// under test; every other method panics via the nil embedded interface, which
// is acceptable because those routes are never exercised here.
type stubRepo struct {
	service.Repository
	sys service.SystemService
}

func (s *stubRepo) System() service.SystemService { return s.sys }

type stubSystem struct {
	service.SystemService
}

func (s *stubSystem) GetDeviceInfo() model.DeviceInfo {
	return model.DeviceInfo{
		LanIpv4:     []string{"192.168.1.10"},
		Port:        80,
		DeviceName:  "casaos-test",
		DeviceModel: "CasaOS",
		DeviceSN:    "SN123456",
		Initialized: true,
		OS_Version:  "0.4.17",
		Hash:        "abc123",
	}
}

// setStubService wires the discovery handlers to a canned DeviceInfo source.
func setStubService(t *testing.T) {
	t.Helper()
	service.MyService = &stubRepo{sys: &stubSystem{}}
}

// ---- helpers -------------------------------------------------------------

func performRouteRequest(router http.Handler, path string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodGet, path, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	return w
}

func decodeBody(t *testing.T, w *httptest.ResponseRecorder) map[string]interface{} {
	t.Helper()
	var body map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("response is not valid JSON: %v (body: %s)", err, w.Body.String())
	}
	return body
}

func assertSuccess200(t *testing.T, body map[string]interface{}) {
	t.Helper()
	success, ok := body["success"]
	if !ok {
		t.Fatalf("response missing \"success\" field: %v", body)
	}
	if success != float64(200) {
		t.Fatalf("expected success == 200, got %v", success)
	}
}

// ---- v1 discovery ---------------------------------------------------------

func TestV1SysInfoDiscovery(t *testing.T) {
	setStubService(t)

	router := InitV1Router()
	w := performRouteRequest(router, "/v1/sys/info")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	assertSuccess200(t, body)

	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected \"data\" object in response, got %v", body["data"])
	}
	if data["os_version"] != "0.4.17" {
		t.Fatalf("expected data.os_version == \"0.4.17\", got %v", data["os_version"])
	}
}

func TestV1SysVersionStillRequiresAuth(t *testing.T) {
	setStubService(t)

	router := InitV1Router()
	w := performRouteRequest(router, "/v1/sys/version")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated /v1/sys/version, got %d (body: %s)", w.Code, w.Body.String())
	}
}

// ---- v2 discovery ---------------------------------------------------------

func TestV2ZimaOSDeviceInfo(t *testing.T) {
	setStubService(t)

	router := InitV2Router()
	w := performRouteRequest(router, "/v2/zimaos/device/info")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	assertSuccess200(t, body)

	if body["os_version"] != "0.4.17" {
		t.Fatalf("expected top-level os_version == \"0.4.17\" (Zima client reads this), got %v", body["os_version"])
	}
	if body["device_name"] != "casaos-test" {
		t.Fatalf("expected top-level device_name == \"casaos-test\", got %v", body["device_name"])
	}
	if _, ok := body["data"].(map[string]interface{}); !ok {
		t.Fatalf("expected \"data\" object in response, got %v", body["data"])
	}
}

func TestV2CasaOSDeviceInfo(t *testing.T) {
	setStubService(t)

	router := InitV2Router()
	w := performRouteRequest(router, "/v2/casaos/device/info")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	assertSuccess200(t, body)

	if _, ok := body["data"].(map[string]interface{}); !ok {
		t.Fatalf("expected \"data\" object in response, got %v", body["data"])
	}
	if body["os_version"] != "0.4.17" {
		t.Fatalf("expected top-level os_version == \"0.4.17\", got %v", body["os_version"])
	}
}

func TestV2SysInfo(t *testing.T) {
	setStubService(t)

	router := InitV2Router()
	w := performRouteRequest(router, "/v2/sys/info")

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body: %s)", w.Code, w.Body.String())
	}

	body := decodeBody(t, w)
	assertSuccess200(t, body)

	data, ok := body["data"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected \"data\" object in response, got %v", body["data"])
	}
	if data["os_version"] != "0.4.17" {
		t.Fatalf("expected data.os_version == \"0.4.17\", got %v", data["os_version"])
	}
}

// ---- security invariants --------------------------------------------------

func TestV2HealthStillRequiresAuth(t *testing.T) {
	setStubService(t)

	router := InitV2Router()
	w := performRouteRequest(router, "/v2/casaos/health/services")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 for unauthenticated /v2/casaos/health/services, got %d (body: %s)", w.Code, w.Body.String())
	}
}

func TestV2UnknownPathStillRequiresAuth(t *testing.T) {
	setStubService(t)

	router := InitV2Router()

	// Any non-whitelisted path must NOT become reachable without a token.
	paths := []string{
		"/v2/casaos/device/nonexistent",
		"/v2/zimaos/device/other",
		"/v2/sys/status",
	}
	for _, p := range paths {
		w := performRouteRequest(router, p)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("expected 401 for unauthenticated %s, got %d (body: %s)", p, w.Code, w.Body.String())
		}
	}
}
