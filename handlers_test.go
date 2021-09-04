package main

import (
	"bytes"
	"encoding/json"

	"github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"runtime"
	"testing"
)

func TestHandler_GetAllProduce(t *testing.T) {
	db := NewDB(logrus.New())
	db.Produce = []*ProduceItem{
		{
			Name:      "carrot",
			Code:      "1234-abcd-ABCD-1234",
			UnitPrice: 2.00,
		},
		{
			Name:      "beet",
			Code:      "3345-abcd-ABCD-1234",
			UnitPrice: 1.00,
		},
	}

	expected, err := json.Marshal(db.Produce)
	if err != nil {
		t.Fatal(err)
	}

	h := NewHandler(db, runtime.NumCPU(), logrus.New())

	req := httptest.NewRequest(http.MethodGet, "/api/v1/Produce/", nil)
	w := httptest.NewRecorder()
	h.GetAllProduce(w, req)
	res := w.Result()
	defer res.Body.Close()
	data, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if string(data) != string(expected) {
		t.Errorf("expected response got %v", string(data))
	}
	w.Flush()

	db.Produce = []*ProduceItem{
		{
			Name:      "salad",
			Code:      "!@#$%",
			UnitPrice: 2.00,
		},
		{
			Name:      "!@#$%",
			Code:      "3345-abcd-ABCD-1234",
			UnitPrice: 1.00,
		},
		{
			Name:      "orange",
			Code:      "1234-abcd-ABCD-1234",
			UnitPrice: 1.000000003,
		},
		{
			Name:      "apple",
			Code:      "1431-abcd-ABCD-1234",
			UnitPrice: 2.00,
		},
	}

	req = httptest.NewRequest(http.MethodGet, "/api/v1/Produce/", nil)
	h.GetAllProduce(w, req)
	res = w.Result()
	defer res.Body.Close()
	data, err = ioutil.ReadAll(res.Body)
	if err != nil {
		t.Errorf("expected error to be nil got %v", err)
	}
	if string(data) != "" {
		t.Errorf("expected response got %v", string(data))
	}
	w.Flush()

}

func Test_Handlers(t *testing.T) {

	logger := logrus.New()
	logger.Level = 1
	var err error
	db, err := LoadDB(logger)
	if err != nil {
		t.Fatal(err)
	}
	h := NewHandler(db, runtime.NumCPU(), logger)

	r := LoadRouter(h)

	ts := httptest.NewServer(r)
	defer ts.Close()

	// Get /api/v1/produce     -- list all produce in db
	expectedBody := `[{"produce_name":"Lettuce","produce_code":"A12T-4GH7-QPL9-3N4M","produce_unit_price":3.46},{"produce_name":"Peach","produce_code":"E5T6-9UI3-TH15-QR88","produce_unit_price":2.99},{"produce_name":"Green Pepper","produce_code":"YRT6-72AS-K736-L4AR","produce_unit_price":0.79},{"produce_name":"Gala Apple","produce_code":"TQ4C-VV6T-75ZX-1RMR","produce_unit_price":3.59}]`
	if rr, body := testRequest(t, ts, "GET", "/api/v1/produce", nil); body != expectedBody {
		t.Logf("%s  ::   %s", rr.Status, body)
		t.Fail()
	}

	expectedBody = `{"produce_name":"Lettuce","produce_code":"A12T-4GH7-QPL9-3N4M","produce_unit_price":3.46}`
	// GET /api/v1/produce/A12T-4GH7-QPL9-3N4M     -- list produce with code A12T-4GH7-QPL9-3N4M
	if rr, body := testRequest(t, ts, "GET", "/api/v1/produce/A12T-4GH7-QPL9-3N4M", nil); body != expectedBody {
		t.Logf("%s  ::   %s", rr.Status, body)
		t.Fail()
	}
	// GET /api/v1/produce/      -- list produce with code that is empty space
	if rr, body := testRequest(t, ts, "GET", "/api/v1/produce/ ", nil); rr.StatusCode != 400 {
		t.Logf("%s  ::   %s", rr.Status, body)
		t.Fail()
	}

	// DELETE /api/v1/produce/A12T-4GH7-QPL9-3N4M - delete existing item
	if rr, body := testRequest(t, ts, "DELETE", "/api/v1/produce/A12T-4GH7-QPL9-3N4M", nil); rr.StatusCode != 204 {
		t.Logf("%s  ::   %s", rr.Status, body)
		t.Fail()
	}

	// GET /api/v1/produce/A12T-4GH7-QPL9-3N4M     -- should not be found this time was deleted in previous test case A12T-4GH7-QPL9-3N4M
	if rr, body := testRequest(t, ts, "GET", "/api/v1/produce/A12T-4GH7-QPL9-3N4M", nil); rr.StatusCode != 404 {
		t.Logf("%s  ::   %s", rr.Status, body)
		t.Fail()
	}

	// DELETE /api/v1/produce/A12T-4GH7-QPL9-3N4M -- delete item that does not exist
	if rr, body := testRequest(t, ts, "DELETE", "/api/v1/produce/A12T-4GH7-QPL9-3N4M", nil); rr.StatusCode != 404 {
		t.Logf("%s  ::   %s", rr.Status, body)
		t.Fail()
	}

	// DELETE /api/v1/produce/ -- attempt to delete without providing code
	if rr, body := testRequest(t, ts, "DELETE", "/api/v1/produce", nil); rr.StatusCode != 405 {
		t.Logf("%s  ::   %s", rr.Status, body)
		t.Fail()
	}

	// ADD /api/v1/produce -- add  items that does not exist, exist,are duplicates, and have invalid names, prices and codes
	payload := `[
		{"produce_name":"Lettuce","produce_code":"A12T-4GH7-QPL9-3N4M","produce_unit_price":3.46},
		{"produce_name":"Peach","produce_code":"E5T6-9UI3-TH15-QR88","produce_unit_price":2.99},
		{"produce_name":"InvalidProduceCode","produce_code":"@12T-4GH7-QPL9-3N4M","produce_unit_price":3.46},
		{"produce_name":"Inv@lidName","produce_code":"A13T-4GH7-QPL9-3N4M","produce_unit_price":3.46},
		{"produce_name":"Lettuce","produce_code":"A15T-4GH7-QPL9-3N4M","produce_unit_price":3.4655555555}],
		{"produce_name":"Lettuce","produce_code":"A16T-4GH7-QPL9-3N4M","produce_unit_price":-3.46}]`
	if rr, body := testRequest(t, ts, "POST", "/api/v1/produce/", bytes.NewBuffer([]byte(payload))); rr.StatusCode == 200 {

		var ar AddResults
		json.Unmarshal([]byte(body), &ar)

		for _, x := range ar.Results {

			switch x.Produce.Code {

			case "A12T-4GH7-QPL9-3N4M":
				if x.StatusCode != 201 && x.Status == "201: added" {
					t.Logf("%s  ::  %s", x.Status, x.Produce.Code)
					t.Fail()
				}
			case "E5T6-9UI3-TH15-QR88":
				if x.StatusCode != 409 && x.Status == "409: item already exists" {
					t.Logf("%s  ::  %s", x.Status, x.Produce.Code)
					t.Fail()
				}
			case "@12T-4GH7-QPL9-3N4M":
				if x.Status != "400 item code is invalid" && x.StatusCode == 400 {
					t.Logf("%s  ::  %s", x.Status, x.Produce.Code)
					t.Fail()
				}
			case "A13T-4GH7-QPL9-3N4M":
				if x.StatusCode != 400 && x.Status == "400: item name is invalid" {
					t.Logf("%s  ::  %s", x.Status, x.Produce.Code)
					t.Fail()
				}

			case "A15T-4GH7-QPL9-3N4M":
				if x.Produce.UnitPrice != 3.47 && x.StatusCode == 201 && x.Status == "201: added" {
					t.Logf("%s  ::  %s", x.Status, x.Produce.Code)
					t.Fail()
				}
			case "A16T-4GH7-QPL9-3N4M":
				if x.StatusCode == 400 && x.Status == "400: unit price is invalid" {
					t.Logf("%s  ::  %s", x.Status, x.Produce.Code)
					t.Fail()
				}

			}

		}

	}

}

// testRequest is straight from the chi library code used in their tests
// https://github.com/go-chi/chi/blob/df44563f0692b1e677f18220b9be165e481cf51b/middleware/middleware_test.go#L83
func testRequest(t *testing.T, ts *httptest.Server, method, path string, body io.Reader) (*http.Response, string) {
	req, err := http.NewRequest(method, ts.URL+path, body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}

	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
		return nil, ""
	}
	defer resp.Body.Close()

	return resp, string(respBody)
}
