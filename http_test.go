package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestPostJSON(t *testing.T) {
	type Req struct {
		A string `json:"a"`
	}
	type Res struct {
		A string `json:"a"`
		B string `json:"b"`
		C int    `json:"c"`
	}

	var (
		req       *Req
		wantReq   string
		resStatus int
		res       *Res
		resBytes  []byte
		wantRes   *Res
	)

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("failed to read request body: %v", err)
		}
		if string(data) != wantReq {
			t.Errorf("unexpected request body. Got %s, want %s", data, wantReq)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(resStatus)
		w.Write(resBytes)
	}))
	defer ts.Close()

	cases := []struct {
		Label   string
		Request Req
		// What we expect the server to receive
		WantReq string
		// Status we expect the server to respond with
		ResStatus int
		// Response body we expect the server to respond with
		ResBytes []byte
		// What we expect ResBytes to be unmarshaled to
		WantRes Res
		// Do we expect an error?
		WantErr bool
	}{
		// Only testing so much here, since this mostly wraps stdlib functions.
		{
			Label:     "happy path",
			Request:   Req{A: "a"},
			WantReq:   `{"a":"a"}`,
			ResStatus: http.StatusOK,
			ResBytes:  []byte(`{"a":"a","b":"b","c":1}`),
			WantRes:   Res{A: "a", B: "b", C: 1},
			WantErr:   false,
		},
		{
			Label:     "non-200 status produces error",
			Request:   Req{A: "a"},
			WantReq:   `{"a":"a"}`,
			ResStatus: http.StatusBadGateway,
			ResBytes:  []byte(``),
			WantRes:   Res{},
			WantErr:   true,
		},
		{
			Label:     "mal-formed response produces error",
			Request:   Req{A: "a"},
			WantReq:   `{"a":"a"}`,
			ResStatus: http.StatusOK,
			ResBytes:  []byte(`{oops}`),
			WantRes:   Res{},
			WantErr:   true,
		},
	}
	for _, c := range cases {
		t.Run(c.Label, func(t *testing.T) {
			req = &c.Request
			wantReq = c.WantReq
			resStatus = c.ResStatus
			resBytes = c.ResBytes
			wantRes = &c.WantRes
			res = &Res{}

			err := PostJSON[Req, Res](ts.URL, nil, req, res)
			if !c.WantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.WantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if res.A != wantRes.A || res.B != wantRes.B || res.C != wantRes.C {
				t.Fatalf("unexpected response. Got %v, want %v", res, wantRes)
			}
		})
	}
}
