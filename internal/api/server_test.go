package api_test

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/better-monitoring/glycoview/internal/config"
	storepkg "github.com/better-monitoring/glycoview/internal/store"
	"github.com/better-monitoring/glycoview/internal/testutil"
)

func TestV1StatusAndEntries(t *testing.T) {
	h := testutil.NewHarness("readable")
	defer h.Close()

	now := time.Now().UTC()
	if err := h.Store.Seed("entries",
		map[string]any{"type": "sgv", "sgv": 111, "date": now.Add(-5 * time.Minute).UnixMilli()},
		map[string]any{"type": "sgv", "sgv": 115, "date": now.UnixMilli()},
	); err != nil {
		t.Fatal(err)
	}

	resp, err := h.Client().Get(h.Server.URL + "/api/v1/status.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status.json = %d", resp.StatusCode)
	}
	var status struct {
		APIEnabled bool `json:"apiEnabled"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if !status.APIEnabled {
		t.Fatalf("apiEnabled = false")
	}

	resp, err = h.Client().Get(h.Server.URL + "/api/v1/entries/current.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var current []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&current); err != nil {
		t.Fatal(err)
	}
	if len(current) != 1 {
		t.Fatalf("current len = %d", len(current))
	}
	if int(current[0]["sgv"].(float64)) != 115 {
		t.Fatalf("current sgv = %v", current[0]["sgv"])
	}
	id := current[0]["_id"].(string)

	resp, err = h.Client().Get(h.Server.URL + "/api/v1/entries/" + id + ".json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var byID []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&byID); err != nil {
		t.Fatal(err)
	}
	if len(byID) != 1 || byID[0]["_id"] != id {
		t.Fatalf("lookup by id failed: %+v", byID)
	}

	resp, err = h.Client().Get(h.Server.URL + "/api/v1/echo/entries/sgv.json?find[sgv][$gte]=100")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("echo status = %d", resp.StatusCode)
	}
}

func TestV1TreatmentsCRUDAndSanitize(t *testing.T) {
	h := testutil.NewHarness("readable")
	defer h.Close()

	subject := h.Auth.CreateSubject("test-api-create", []string{"apiCreate", "apiRead", "apiDelete"})
	createdAt := time.Now().Add(-2 * time.Hour).UTC().Format("2006-01-02T15:04:05.000Z")
	body := `{"eventType":"Meal Bolus","created_at":"` + createdAt + `","carbs":"30","insulin":"2.0","notes":"<IMG SRC=\"javascript:alert('XSS');\">"}`
	req, err := http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/treatments", strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("api-secret", subject.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("post treatments status = %d", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/treatments.json?find[carbs]=30", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("api-secret", subject.AccessToken)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var treatments []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&treatments); err != nil {
		t.Fatal(err)
	}
	if len(treatments) != 1 {
		t.Fatalf("treatments len = %d", len(treatments))
	}
	if treatments[0]["notes"] != "<img>" {
		t.Fatalf("notes = %v", treatments[0]["notes"])
	}

	req, err = http.NewRequest(http.MethodDelete, h.Server.URL+"/api/v1/treatments?find[carbs]=30", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("api-secret", subject.AccessToken)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete treatments status = %d", resp.StatusCode)
	}
}

func TestV1SecurityModes(t *testing.T) {
	h := testutil.NewHarness("denied")
	defer h.Close()

	if err := h.Store.Seed("entries", map[string]any{"type": "sgv", "sgv": 120, "date": time.Now().UnixMilli()}); err != nil {
		t.Fatal(err)
	}

	resp, err := h.Client().Get(h.Server.URL + "/api/v1/entries.json")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth status = %d", resp.StatusCode)
	}

	sum := sha1.Sum([]byte(h.Config.APISecret))
	req, err := http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/entries.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("api-secret", hex.EncodeToString(sum[:]))
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("api-secret status = %d", resp.StatusCode)
	}

	subject := h.Auth.CreateSubject("test-reader", []string{"apiRead"})
	resp, err = h.Client().Get(h.Server.URL + "/api/v2/authorization/request/" + subject.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var tokenResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		t.Fatal(err)
	}
	req, err = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/entries.json", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+tokenResponse["token"])
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("bearer status = %d", resp.StatusCode)
	}

	resp, err = h.Client().Get(h.Server.URL + "/api/v1/verifyauth")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var verify map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&verify); err != nil {
		t.Fatal(err)
	}
	message := verify["message"].(map[string]any)
	if message["message"] != "UNAUTHORIZED" {
		t.Fatalf("verifyauth message = %v", message["message"])
	}
}

func TestV3Workflow(t *testing.T) {
	h := testutil.NewHarness("denied")
	defer h.Close()

	subject := h.Auth.CreateSubject("test-api-all", []string{"apiCreate", "apiRead", "apiUpdate", "apiDelete"})
	resp, err := h.Client().Get(h.Server.URL + "/api/v2/authorization/request/" + subject.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var tokenResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		t.Fatal(err)
	}
	jwt := tokenResponse["token"]

	historyStart := time.Now().Add(-time.Minute).UnixMilli()
	body := `{"eventType":"Correction Bolus","insulin":1,"date":1760000000000,"app":"test","device":"go-suite"}`
	req, _ := http.NewRequest(http.MethodPost, h.Server.URL+"/api/v3/treatments", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create status = %d", resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	identifier := created["identifier"].(string)

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/treatments/"+identifier, nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("read status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPatch, h.Server.URL+"/api/v3/treatments/"+identifier, strings.NewReader(`{"carbs":5,"insulin":0.4}`))
	req.Header.Set("Authorization", "Bearer "+jwt)
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/treatments?identifier$eq="+identifier, nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var search struct {
		Result []map[string]any `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&search); err != nil {
		t.Fatal(err)
	}
	if len(search.Result) != 1 {
		t.Fatalf("search len = %d", len(search.Result))
	}

	req, _ = http.NewRequest(http.MethodDelete, h.Server.URL+"/api/v3/treatments/"+identifier, nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/treatments/history/"+strconv.FormatInt(historyStart, 10), nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("history status = %d", resp.StatusCode)
	}
}

func TestV1EntriesPostDeleteAndDevicestatus(t *testing.T) {
	h := testutil.NewHarness("denied")
	defer h.Close()

	subject := h.Auth.CreateSubject("test-writer", []string{"apiCreate", "apiRead", "apiDelete"})

	req, _ := http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/entries", strings.NewReader(`[{"type":"sgv","sgv":"199","dateString":"2014-07-20T00:44:15.000-07:00","date":1405791855000,"device":"dexcom"},{"type":"sgv","sgv":"200","dateString":"2014-07-20T00:44:15.001-07:00","date":1405791855001,"device":"dexcom"}]`))
	req.Header.Set("api-secret", subject.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("post entries status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/entries.json?find[dateString][$gte]=2014-07-20&count=100", nil)
	req.Header.Set("api-secret", subject.AccessToken)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var entries []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries len = %d", len(entries))
	}
	if int(entries[0]["utcOffset"].(float64)) != -420 {
		t.Fatalf("entries utcOffset = %v", entries[0]["utcOffset"])
	}

	req, _ = http.NewRequest(http.MethodDelete, h.Server.URL+"/api/v1/entries.json?find[dateString][$gte]=2014-07-20&count=100", nil)
	req.Header.Set("api-secret", subject.AccessToken)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("delete entries status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/devicestatus", strings.NewReader(`{"device":"xdripjs://rigName","xdripjs":{"state":6},"created_at":"2018-12-16T01:00:52Z"}`))
	req.Header.Set("api-secret", subject.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("post devicestatus status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v1/devicestatus.json?find[created_at][$gte]=2018-12-16&find[created_at][$lte]=2018-12-17", nil)
	req.Header.Set("api-secret", subject.AccessToken)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var statuses []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&statuses); err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 1 {
		t.Fatalf("devicestatus len = %d", len(statuses))
	}
}

func TestV1EntriesPreviewAndUnauthorizedWrite(t *testing.T) {
	h := testutil.NewHarness("readable")
	defer h.Close()

	req, _ := http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/entries.json", strings.NewReader(`[{"type":"sgv","sgv":100,"date":1760000000000,"dateString":"2025-10-09T10:00:00Z"}]`))
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauthorized entry post status = %d", resp.StatusCode)
	}

	subject := h.Auth.CreateSubject("test-entry-preview", []string{"apiCreate"})
	req, _ = http.NewRequest(http.MethodPost, h.Server.URL+"/api/v1/entries/preview.json", strings.NewReader(`[{"type":"sgv","sgv":100,"date":1760000000000,"dateString":"2025-10-09T10:00:00Z"}]`))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-secret", subject.AccessToken)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("preview status = %d", resp.StatusCode)
	}
	var preview []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&preview); err != nil {
		t.Fatal(err)
	}
	if len(preview) != 1 {
		t.Fatalf("preview len = %d", len(preview))
	}
}

func TestV1VersionsSliceAndTimes(t *testing.T) {
	h := testutil.NewHarness("readable")
	defer h.Close()

	for i := 0; i < 20; i++ {
		ts := time.Date(2014, 7, 20, 0, i%10, 0, 0, time.UTC).UnixMilli() + int64(i)
		if err := h.Store.Seed("entries", map[string]any{"type": "sgv", "sgv": 150 + i, "date": ts, "dateString": time.UnixMilli(ts).UTC().Format("2006-01-02T15:04:05")}); err != nil {
			t.Fatal(err)
		}
	}

	resp, err := h.Client().Get(h.Server.URL + "/api/versions")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("versions status = %d", resp.StatusCode)
	}

	resp, err = h.Client().Get(h.Server.URL + "/api/v1/slice/entries/dateString/sgv/2014-07.json?count=20")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var sliced []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&sliced); err != nil {
		t.Fatal(err)
	}
	if len(sliced) != 20 {
		t.Fatalf("slice len = %d", len(sliced))
	}

	resp, err = h.Client().Get(h.Server.URL + "/api/v1/times/echo/2014-07/.*T{00..05}:.json?count=20&find[sgv][$gte]=160")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var echoed map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&echoed); err != nil {
		t.Fatal(err)
	}
	patterns := echoed["pattern"].([]any)
	if len(patterns) != 6 {
		t.Fatalf("times echo patterns = %d", len(patterns))
	}
}

func TestV3SearchValidationAndProjection(t *testing.T) {
	h := testutil.NewHarness("denied")
	defer h.Close()

	subject := h.Auth.CreateSubject("test-api-all-search", []string{"apiAll"})
	resp, err := h.Client().Get(h.Server.URL + "/api/v2/authorization/request/" + subject.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var tokenResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		t.Fatal(err)
	}
	jwt := tokenResponse["token"]

	for i := 0; i < 10; i++ {
		body := `{"type":"sgv","date":176000000000` + strconv.Itoa(i) + `,"app":"test","device":"search-suite","sgv":` + strconv.Itoa(100+i) + `}`
		req, _ := http.NewRequest(http.MethodPost, h.Server.URL+"/api/v3/entries", strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer "+jwt)
		req.Header.Set("Content-Type", "application/json")
		resp, err = h.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
	}

	req, _ := http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/entries?limit=INVALID", nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("invalid limit status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/entries?sort=date&sort$desc=created_at", nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("sort conflict status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/entries?fields=date,app,subject&limit=3", nil)
	req.Header.Set("Authorization", "Bearer "+jwt)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var projected struct {
		Result []map[string]any `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&projected); err != nil {
		t.Fatal(err)
	}
	if len(projected.Result) != 3 {
		t.Fatalf("projected len = %d", len(projected.Result))
	}
	if _, exists := projected.Result[0]["identifier"]; exists {
		t.Fatalf("identifier unexpectedly present in projected result")
	}
}

func TestV3UpdateGuards(t *testing.T) {
	h := testutil.NewHarness("denied")
	defer h.Close()

	all := h.Auth.CreateSubject("test-api-all-guards", []string{"apiCreate", "apiRead", "apiUpdate", "apiDelete"})
	read := h.Auth.CreateSubject("test-api-read-guards", []string{"apiRead"})

	tokenResp, err := h.Client().Get(h.Server.URL + "/api/v2/authorization/request/" + all.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	defer tokenResp.Body.Close()
	var allJWT map[string]string
	if err := json.NewDecoder(tokenResp.Body).Decode(&allJWT); err != nil {
		t.Fatal(err)
	}
	tokenResp, err = h.Client().Get(h.Server.URL + "/api/v2/authorization/request/" + read.AccessToken)
	if err != nil {
		t.Fatal(err)
	}
	defer tokenResp.Body.Close()
	var readJWT map[string]string
	if err := json.NewDecoder(tokenResp.Body).Decode(&readJWT); err != nil {
		t.Fatal(err)
	}

	req, _ := http.NewRequest(http.MethodPost, h.Server.URL+"/api/v3/treatments", strings.NewReader(`{"identifier":"guard-doc","date":1760000100000,"utcOffset":120,"app":"test","device":"guard","eventType":"Correction Bolus","insulin":0.3}`))
	req.Header.Set("Authorization", "Bearer "+allJWT["token"])
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create guard doc status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPut, h.Server.URL+"/api/v3/treatments/new-upsert", strings.NewReader(`{"identifier":"new-upsert","date":1760000200000,"utcOffset":120,"app":"test","device":"guard","eventType":"Correction Bolus","insulin":0.3}`))
	req.Header.Set("Authorization", "Bearer "+readJWT["token"])
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("upsert without create status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPut, h.Server.URL+"/api/v3/treatments/guard-doc", strings.NewReader(`{"identifier":"guard-doc","date":1760000100001,"utcOffset":120,"app":"test","device":"guard","eventType":"Correction Bolus","insulin":0.4}`))
	req.Header.Set("Authorization", "Bearer "+allJWT["token"])
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("date mutation status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPut, h.Server.URL+"/api/v3/treatments/guard-doc", strings.NewReader(`{"identifier":"guard-doc","date":1760000100000,"utcOffset":120,"app":"test","device":"guard","eventType":"Correction Bolus","carbs":5}`))
	req.Header.Set("Authorization", "Bearer "+allJWT["token"])
	req.Header.Set("If-Unmodified-Since", time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat))
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPreconditionFailed {
		t.Fatalf("if-unmodified-since status = %d", resp.StatusCode)
	}
}

func TestV3SecurityReadAndDeleteParity(t *testing.T) {
	h := testutil.NewHarness("denied")
	defer h.Close()

	resp, err := h.Client().Get(h.Server.URL + "/api/v3/test")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("missing auth status = %d", resp.StatusCode)
	}
	var missing map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&missing); err != nil {
		t.Fatal(err)
	}
	if missing["message"] != "Missing or bad access token or JWT" {
		t.Fatalf("missing auth message = %v", missing["message"])
	}

	req, _ := http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/test", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("bad token status = %d", resp.StatusCode)
	}
	var bad map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&bad); err != nil {
		t.Fatal(err)
	}
	if bad["message"] != "Bad access token or JWT" {
		t.Fatalf("bad token message = %v", bad["message"])
	}

	deniedJWT := issueJWT(t, h, h.Auth.CreateSubject("test-api-denied-v3", []string{"denied"}).AccessToken)
	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/test", nil)
	req.Header.Set("Authorization", "Bearer "+deniedJWT)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("denied status = %d", resp.StatusCode)
	}
	var forbidden map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&forbidden); err != nil {
		t.Fatal(err)
	}
	if forbidden["message"] != "Missing permission api:entries:read" {
		t.Fatalf("denied message = %v", forbidden["message"])
	}

	req, _ = http.NewRequest(http.MethodPost, h.Server.URL+"/api/v3/devicestatus", strings.NewReader(`{"date":1760000300000,"app":"test","device":"read-parity","uploaderBattery":58}`))
	req.Header.Set("api-secret", h.Config.APISecret)
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create devicestatus status = %d", resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	identifier := created["identifier"].(string)

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/devicestatus/"+identifier+"?fields=date,device,subject", nil)
	req.Header.Set("api-secret", h.Config.APISecret)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var projected map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&projected); err != nil {
		t.Fatal(err)
	}
	result := projected["result"].(map[string]any)
	if len(result) != 3 {
		t.Fatalf("projected fields = %+v", result)
	}
	if _, exists := result["_id"]; exists {
		t.Fatalf("_id unexpectedly present in projected fields")
	}

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/devicestatus/"+identifier+"?fields=_all", nil)
	req.Header.Set("api-secret", h.Config.APISecret)
	req.Header.Set("If-Modified-Since", time.Now().Add(time.Second).UTC().Format(http.TimeFormat))
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotModified {
		t.Fatalf("if-modified-since status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/devicestatus/"+identifier+"?fields=_all", nil)
	req.Header.Set("api-secret", h.Config.APISecret)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var full map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&full); err != nil {
		t.Fatal(err)
	}
	fullResult := full["result"].(map[string]any)
	if _, exists := fullResult["_id"]; exists {
		t.Fatalf("_id unexpectedly present in full result")
	}
	if fullResult["identifier"] != identifier {
		t.Fatalf("identifier = %v", fullResult["identifier"])
	}

	req, _ = http.NewRequest(http.MethodDelete, h.Server.URL+"/api/v3/devicestatus/"+identifier, nil)
	req.Header.Set("api-secret", h.Config.APISecret)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("soft delete status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/devicestatus/"+identifier, nil)
	req.Header.Set("api-secret", h.Config.APISecret)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusGone {
		t.Fatalf("soft-deleted read status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPost, h.Server.URL+"/api/v3/devicestatus", strings.NewReader(`{"date":1760000300000,"app":"test","device":"read-parity","uploaderBattery":60}`))
	req.Header.Set("api-secret", h.Config.APISecret)
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("restore deleted status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodDelete, h.Server.URL+"/api/v3/devicestatus/"+identifier+"?permanent=true", nil)
	req.Header.Set("api-secret", h.Config.APISecret)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("permanent delete status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/devicestatus/"+identifier, nil)
	req.Header.Set("api-secret", h.Config.APISecret)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("permanent-deleted read status = %d", resp.StatusCode)
	}
}

func TestV3CreateDeduplicationAndPatchMetadata(t *testing.T) {
	h := testutil.NewHarness("denied")
	defer h.Close()

	createJWT := issueJWT(t, h, h.Auth.CreateSubject("test-api-create-v3", []string{"apiCreate", "apiRead"}).AccessToken)
	updateJWT := issueJWT(t, h, h.Auth.CreateSubject("test-api-update-v3", []string{"apiUpdate", "apiRead", "apiDelete"}).AccessToken)

	doc := `{"date":1760000400000,"utcOffset":120,"app":"test","device":"dedupe-suite","eventType":"Correction Bolus","insulin":0.3}`
	req, _ := http.NewRequest(http.MethodPost, h.Server.URL+"/api/v3/treatments", strings.NewReader(doc))
	req.Header.Set("Authorization", "Bearer "+createJWT)
	req.Header.Set("Content-Type", "application/json")
	resp, err := h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create treatment status = %d", resp.StatusCode)
	}
	var created map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	identifier := created["identifier"].(string)
	expectedIdentifier := storepkg.CalculateIdentifier(map[string]any{
		"date":      int64(1760000400000),
		"device":    "dedupe-suite",
		"eventType": "Correction Bolus",
	})
	if identifier != expectedIdentifier {
		t.Fatalf("identifier = %s, want %s", identifier, expectedIdentifier)
	}

	req, _ = http.NewRequest(http.MethodPatch, h.Server.URL+"/api/v3/treatments/"+identifier, strings.NewReader(`{"carbs":10}`))
	req.Header.Set("Authorization", "Bearer "+updateJWT)
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("patch status = %d", resp.StatusCode)
	}
	var patched map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&patched); err != nil {
		t.Fatal(err)
	}
	patchedResult := patched["result"].(map[string]any)
	if patchedResult["subject"] != "test-api-create-v3" {
		t.Fatalf("subject = %v", patchedResult["subject"])
	}
	if patchedResult["modifiedBy"] != "test-api-update-v3" {
		t.Fatalf("modifiedBy = %v", patchedResult["modifiedBy"])
	}

	req, _ = http.NewRequest(http.MethodPost, h.Server.URL+"/api/v3/treatments", strings.NewReader(doc))
	req.Header.Set("Authorization", "Bearer "+updateJWT)
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("dedupe post status = %d", resp.StatusCode)
	}
	var dedupe map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&dedupe); err != nil {
		t.Fatal(err)
	}
	if dedupe["isDeduplication"] != true {
		t.Fatalf("isDeduplication = %v", dedupe["isDeduplication"])
	}

	req, _ = http.NewRequest(http.MethodPatch, h.Server.URL+"/api/v3/treatments/"+identifier, strings.NewReader(`{"identifier":"MODIFIED"}`))
	req.Header.Set("Authorization", "Bearer "+updateJWT)
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("patch immutable identifier status = %d", resp.StatusCode)
	}

	req, _ = http.NewRequest(http.MethodPost, h.Server.URL+"/api/v3/treatments", strings.NewReader(`{"date":"2019-06-10T08:07:08,576+02:00","app":"test","device":"date-normalize","eventType":"Correction Bolus","insulin":0.4}`))
	req.Header.Set("api-secret", h.Config.APISecret)
	req.Header.Set("Content-Type", "application/json")
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("normalized date create status = %d", resp.StatusCode)
	}
	var normalized map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&normalized); err != nil {
		t.Fatal(err)
	}
	normID := normalized["identifier"].(string)

	req, _ = http.NewRequest(http.MethodGet, h.Server.URL+"/api/v3/treatments/"+normID, nil)
	req.Header.Set("api-secret", h.Config.APISecret)
	resp, err = h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var normalizedDoc map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&normalizedDoc); err != nil {
		t.Fatal(err)
	}
	normResult := normalizedDoc["result"].(map[string]any)
	if int(normResult["utcOffset"].(float64)) != 120 {
		t.Fatalf("utcOffset = %v", normResult["utcOffset"])
	}
	if int64(normResult["date"].(float64)) != 1560146828576 {
		t.Fatalf("date = %v", normResult["date"])
	}
}

func issueJWT(t *testing.T, h *testutil.Harness, accessToken string) string {
	t.Helper()
	resp, err := h.Client().Get(h.Server.URL + "/api/v2/authorization/request/" + accessToken)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var tokenResponse map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&tokenResponse); err != nil {
		t.Fatal(err)
	}
	return tokenResponse["token"]
}

func TestUIShellAndOverviewEndpoint(t *testing.T) {
	h := testutil.NewHarness("readable")
	defer h.Close()

	now := time.Now().UTC()
	if err := h.Store.Seed("entries",
		map[string]any{"type": "sgv", "sgv": 112, "date": now.Add(-15 * time.Minute).UnixMilli(), "direction": "Flat"},
		map[string]any{"type": "sgv", "sgv": 118, "date": now.Add(-10 * time.Minute).UnixMilli(), "direction": "FortyFiveUp"},
		map[string]any{"type": "sgv", "sgv": 124, "date": now.Add(-5 * time.Minute).UnixMilli(), "direction": "SingleUp"},
	); err != nil {
		t.Fatal(err)
	}
	if err := h.Store.Seed("treatments",
		map[string]any{"eventType": "Meal Bolus", "created_at": now.Add(-20 * time.Minute).Format("2006-01-02T15:04:05.000Z"), "carbs": 32, "insulin": 3.4},
	); err != nil {
		t.Fatal(err)
	}
	if err := h.Store.Seed("devicestatus",
		map[string]any{"device": "dexcom-g7", "created_at": now.Add(-4 * time.Minute).Format("2006-01-02T15:04:05.000Z"), "uploaderBattery": 78},
	); err != nil {
		t.Fatal(err)
	}

	resp, err := h.Client().Get(h.Server.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ui shell status = %d", resp.StatusCode)
	}
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Fatalf("ui shell content-type = %s", contentType)
	}

	resp, err = h.Client().Get(h.Server.URL + "/app/api/overview")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("ui overview status = %d", resp.StatusCode)
	}
	var overview struct {
		PatientName string `json:"patientName"`
		Current     struct {
			Value string `json:"value"`
		} `json:"current"`
		Metrics []map[string]any `json:"metrics"`
		Devices []map[string]any `json:"devices"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&overview); err != nil {
		t.Fatal(err)
	}
	if overview.Current.Value == "" || overview.Current.Value == "No data" {
		t.Fatalf("ui overview current value = %q", overview.Current.Value)
	}
	if len(overview.Metrics) == 0 {
		t.Fatalf("ui overview metrics missing")
	}
	if len(overview.Devices) == 0 {
		t.Fatalf("ui overview devices missing")
	}
}

func TestDashboardAuthSetupLoginAndRoles(t *testing.T) {
	h := testutil.NewHarness("denied")
	defer h.Close()

	now := time.Now().UTC()
	if err := h.Store.Seed("entries", map[string]any{"type": "sgv", "sgv": 126, "date": now.UnixMilli()}); err != nil {
		t.Fatal(err)
	}

	resp, err := h.Client().Get(h.Server.URL + "/app/api/auth/status")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var status map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		t.Fatal(err)
	}
	if status["setupRequired"] != true {
		t.Fatalf("setupRequired = %v", status["setupRequired"])
	}

	setupResp := doJSON(t, h, http.MethodPost, "/app/api/auth/setup", `{"username":"admin","password":"password123","displayName":"Clinic Admin"}`, "")
	adminCookie := sessionCookie(t, setupResp)
	var setupBody map[string]any
	if err := json.NewDecoder(setupResp.Body).Decode(&setupBody); err != nil {
		t.Fatal(err)
	}
	_ = setupResp.Body.Close()
	user := setupBody["user"].(map[string]any)
	if user["role"] != "admin" {
		t.Fatalf("setup role = %v", user["role"])
	}
	apiSecret, ok := setupBody["apiSecret"].(string)
	if !ok || apiSecret == "" {
		t.Fatalf("setup apiSecret = %v", setupBody["apiSecret"])
	}

	authResp := doJSON(t, h, http.MethodGet, "/app/api/auth/status", "", adminCookie)
	var authBody map[string]any
	if err := json.NewDecoder(authResp.Body).Decode(&authBody); err != nil {
		t.Fatal(err)
	}
	_ = authResp.Body.Close()
	if authBody["authenticated"] != true {
		t.Fatalf("authenticated = %v", authBody["authenticated"])
	}

	overviewResp := doJSON(t, h, http.MethodGet, "/app/api/overview", "", adminCookie)
	if overviewResp.StatusCode != http.StatusOK {
		t.Fatalf("overview with session = %d", overviewResp.StatusCode)
	}
	_ = overviewResp.Body.Close()

	req, err := http.NewRequest(http.MethodGet, h.Server.URL+"/app/api/overview", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("api-secret", apiSecret)
	secretOverviewResp, err := h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if secretOverviewResp.StatusCode != http.StatusOK {
		t.Fatalf("overview with api secret = %d", secretOverviewResp.StatusCode)
	}
	_ = secretOverviewResp.Body.Close()

	secretResp := doJSON(t, h, http.MethodGet, "/app/api/auth/install-secret", "", adminCookie)
	var secretBody map[string]any
	if err := json.NewDecoder(secretResp.Body).Decode(&secretBody); err != nil {
		t.Fatal(err)
	}
	_ = secretResp.Body.Close()
	if secretBody["apiSecret"] != apiSecret {
		t.Fatalf("install secret = %v", secretBody["apiSecret"])
	}

	createResp := doJSON(t, h, http.MethodPost, "/app/api/users", `{"username":"doctor","password":"password123","displayName":"Doctor One","role":"doctor"}`, adminCookie)
	var createBody map[string]any
	if err := json.NewDecoder(createResp.Body).Decode(&createBody); err != nil {
		t.Fatal(err)
	}
	_ = createResp.Body.Close()
	doctor := createBody["user"].(map[string]any)
	doctorID := doctor["id"].(string)

	listResp := doJSON(t, h, http.MethodGet, "/app/api/users", "", adminCookie)
	var listBody map[string]any
	if err := json.NewDecoder(listResp.Body).Decode(&listBody); err != nil {
		t.Fatal(err)
	}
	_ = listResp.Body.Close()
	users := listBody["users"].([]any)
	if len(users) != 2 {
		t.Fatalf("users len = %d", len(users))
	}

	logoutResp := doJSON(t, h, http.MethodPost, "/app/api/auth/logout", `{}`, adminCookie)
	_ = logoutResp.Body.Close()

	loginResp := doJSON(t, h, http.MethodPost, "/app/api/auth/login", `{"username":"doctor","password":"password123"}`, "")
	doctorCookie := sessionCookie(t, loginResp)
	_ = loginResp.Body.Close()

	doctorOverview := doJSON(t, h, http.MethodGet, "/app/api/overview", "", doctorCookie)
	if doctorOverview.StatusCode != http.StatusOK {
		t.Fatalf("doctor overview status = %d", doctorOverview.StatusCode)
	}
	_ = doctorOverview.Body.Close()

	doctorUsers := doJSON(t, h, http.MethodGet, "/app/api/users", "", doctorCookie)
	if doctorUsers.StatusCode != http.StatusForbidden {
		t.Fatalf("doctor users status = %d", doctorUsers.StatusCode)
	}
	_ = doctorUsers.Body.Close()

	adminLogin := doJSON(t, h, http.MethodPost, "/app/api/auth/login", `{"username":"admin","password":"password123"}`, "")
	adminCookie = sessionCookie(t, adminLogin)
	_ = adminLogin.Body.Close()

	patchResp := doJSON(t, h, http.MethodPatch, "/app/api/users/"+doctorID, `{"active":false}`, adminCookie)
	if patchResp.StatusCode != http.StatusOK {
		t.Fatalf("disable doctor status = %d", patchResp.StatusCode)
	}
	_ = patchResp.Body.Close()

	doctorOverview = doJSON(t, h, http.MethodGet, "/app/api/overview", "", doctorCookie)
	if doctorOverview.StatusCode != http.StatusUnauthorized {
		t.Fatalf("disabled doctor overview status = %d", doctorOverview.StatusCode)
	}
	_ = doctorOverview.Body.Close()
}

func TestDashboardSettingsProxyAndRoles(t *testing.T) {
	var configuredTLS map[string]any
	var appliedUpdate map[string]any

	agent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/v1/system/status":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"service":           "glycoview-agent",
				"dockerManaged":     true,
				"stackName":         "glycoview",
				"stackFile":         "/opt/glycoview/stack/stack.yml",
				"stackEnvFile":      "/opt/glycoview/stack/.env",
				"currentTag":        "v1.2.3",
				"currentImage":      "ghcr.io/glycoview/glycoview:v1.2.3",
				"currentAgentTag":   "v1.2.3",
				"currentAgentImage": "ghcr.io/glycoview/glycoview-agent:v1.2.3",
				"tls": map[string]any{
					"domain":        "glycoview.example.com",
					"email":         "admin@example.com",
					"challengeType": "http-01",
				},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/update/check":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"currentTag":      "v1.2.3",
				"latestTag":       "v1.2.4",
				"updateAvailable": true,
				"releaseUrl":      "https://example.com/release",
				"source":          "github-releases",
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/update/apply":
			if err := json.NewDecoder(r.Body).Decode(&appliedUpdate); err != nil {
				t.Fatalf("decode apply update: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "message": "Update applied", "currentTag": appliedUpdate["tag"]})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/update/rollback":
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "message": "Rollback applied", "currentTag": "v1.2.2"})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/tls/providers":
			_ = json.NewEncoder(w).Encode(map[string]any{"providers": []map[string]any{
				{"id": "cloudflare", "label": "Cloudflare", "fields": []map[string]any{{"key": "CF_DNS_API_TOKEN", "label": "API token", "secret": true}}},
			}})
		case r.Method == http.MethodGet && r.URL.Path == "/v1/tls/config":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"domain":        "glycoview.example.com",
				"email":         "admin@example.com",
				"challengeType": "dns-01",
				"provider":      "cloudflare",
				"env":           map[string]any{"CF_DNS_API_TOKEN": "secret-token"},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/v1/tls/configure":
			if err := json.NewDecoder(r.Body).Decode(&configuredTLS); err != nil {
				t.Fatalf("decode configure tls: %v", err)
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"status": "ok", "message": "TLS configuration applied"})
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer agent.Close()

	cfg := config.Config{
		APISecret:    "this is my long pass phrase",
		JWTSecret:    "this is my long pass phrase",
		Enable:       []string{"careportal", "rawbg", "api"},
		DefaultRoles: []string{"denied"},
		API3MaxLimit: 1000,
		AgentURL:     agent.URL,
	}
	h := testutil.NewHarnessWithConfig(cfg)
	defer h.Close()

	setupResp := doJSON(t, h, http.MethodPost, "/app/api/auth/setup", `{"username":"admin","password":"password123","displayName":"Clinic Admin"}`, "")
	adminCookie := sessionCookie(t, setupResp)
	_ = setupResp.Body.Close()

	createResp := doJSON(t, h, http.MethodPost, "/app/api/users", `{"username":"doctor","password":"password123","displayName":"Doctor One","role":"doctor"}`, adminCookie)
	_ = createResp.Body.Close()
	logoutResp := doJSON(t, h, http.MethodPost, "/app/api/auth/logout", `{}`, adminCookie)
	_ = logoutResp.Body.Close()
	loginResp := doJSON(t, h, http.MethodPost, "/app/api/auth/login", `{"username":"doctor","password":"password123"}`, "")
	doctorCookie := sessionCookie(t, loginResp)
	_ = loginResp.Body.Close()
	adminLogin := doJSON(t, h, http.MethodPost, "/app/api/auth/login", `{"username":"admin","password":"password123"}`, "")
	adminCookie = sessionCookie(t, adminLogin)
	_ = adminLogin.Body.Close()

	statusResp := doJSON(t, h, http.MethodGet, "/app/api/settings/status", "", adminCookie)
	if statusResp.StatusCode != http.StatusOK {
		t.Fatalf("settings status = %d", statusResp.StatusCode)
	}
	var statusBody map[string]any
	if err := json.NewDecoder(statusResp.Body).Decode(&statusBody); err != nil {
		t.Fatal(err)
	}
	_ = statusResp.Body.Close()
	if statusBody["currentTag"] != "v1.2.3" {
		t.Fatalf("currentTag = %v", statusBody["currentTag"])
	}

	updateResp := doJSON(t, h, http.MethodPost, "/app/api/settings/updates/apply", `{"tag":"v1.2.4","includeAgent":true}`, adminCookie)
	if updateResp.StatusCode != http.StatusOK {
		t.Fatalf("apply update status = %d", updateResp.StatusCode)
	}
	_ = updateResp.Body.Close()
	if appliedUpdate["tag"] != "v1.2.4" || appliedUpdate["includeAgent"] != true {
		t.Fatalf("applied update payload = %+v", appliedUpdate)
	}

	tlsResp := doJSON(t, h, http.MethodPost, "/app/api/settings/tls/configure", `{"domain":"glycoview.example.com","email":"admin@example.com","challengeType":"dns-01","provider":"cloudflare","env":{"CF_DNS_API_TOKEN":"secret-token"}}`, adminCookie)
	if tlsResp.StatusCode != http.StatusOK {
		t.Fatalf("configure tls status = %d", tlsResp.StatusCode)
	}
	_ = tlsResp.Body.Close()
	if configuredTLS["provider"] != "cloudflare" {
		t.Fatalf("configured tls payload = %+v", configuredTLS)
	}

	doctorResp := doJSON(t, h, http.MethodGet, "/app/api/settings/status", "", doctorCookie)
	if doctorResp.StatusCode != http.StatusForbidden {
		t.Fatalf("doctor settings status = %d", doctorResp.StatusCode)
	}
	_ = doctorResp.Body.Close()
}

func doJSON(t *testing.T, h *testutil.Harness, method, path, body, cookie string) *http.Response {
	t.Helper()
	var reader *bytes.Reader
	if body == "" {
		reader = bytes.NewReader(nil)
	} else {
		reader = bytes.NewReader([]byte(body))
	}
	req, err := http.NewRequest(method, h.Server.URL+path, reader)
	if err != nil {
		t.Fatal(err)
	}
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := h.Client().Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func sessionCookie(t *testing.T, resp *http.Response) string {
	t.Helper()
	for _, cookie := range resp.Cookies() {
		if cookie.Name == "glycoview_session" {
			return cookie.Name + "=" + cookie.Value
		}
	}
	t.Fatalf("missing session cookie")
	return ""
}
