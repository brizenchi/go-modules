package httpresp

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newTestCtx() (*gin.Context, *httptest.ResponseRecorder) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Request = httptest.NewRequest("GET", "/", nil)
	return c, w
}

func decode(t *testing.T, body string) Envelope {
	t.Helper()
	var e Envelope
	if err := json.Unmarshal([]byte(body), &e); err != nil {
		t.Fatalf("decode: %v body=%s", err, body)
	}
	return e
}

func TestOK(t *testing.T) {
	c, w := newTestCtx()
	OK(c, map[string]any{"x": 1})
	if w.Code != 200 {
		t.Errorf("http status = %d, want 200", w.Code)
	}
	env := decode(t, w.Body.String())
	if env.Code != CodeOK || env.Msg != "ok" {
		t.Errorf("envelope = %+v", env)
	}
}

func TestErrorHelpersStatusAndCode(t *testing.T) {
	cases := []struct {
		name     string
		fn       func(*gin.Context, string)
		wantHTTP int
		wantCode int
	}{
		{"BadRequest", BadRequest, 400, CodeBadRequest},
		{"Unauthorized", Unauthorized, 401, CodeUnauthorized},
		{"Forbidden", Forbidden, 403, CodeForbidden},
		{"NotFound", NotFound, 404, CodeNotFound},
		{"Conflict", Conflict, 409, CodeConflict},
		{"TooManyRequests", TooManyRequests, 429, CodeTooManyReq},
		{"InternalError", InternalError, 500, CodeInternal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c, w := newTestCtx()
			tc.fn(c, "msg")
			if w.Code != tc.wantHTTP {
				t.Errorf("http = %d, want %d", w.Code, tc.wantHTTP)
			}
			if !c.IsAborted() {
				t.Error("expected context abort")
			}
			env := decode(t, w.Body.String())
			if env.Code != tc.wantCode {
				t.Errorf("envelope code = %d, want %d", env.Code, tc.wantCode)
			}
			if env.Msg != "msg" {
				t.Errorf("msg = %q", env.Msg)
			}
		})
	}
}

func TestOKWith_SoftError(t *testing.T) {
	c, w := newTestCtx()
	OKWith(c, 1001, "field invalid", gin.H{"field": "email"})
	if w.Code != 200 {
		t.Errorf("http = %d, want 200 for soft error", w.Code)
	}
	env := decode(t, w.Body.String())
	if env.Code != 1001 || !strings.Contains(env.Msg, "field invalid") {
		t.Errorf("envelope = %+v", env)
	}
}

func TestCustom(t *testing.T) {
	c, w := newTestCtx()
	Custom(c, 418, 9999, "i am a teapot", nil)
	if w.Code != 418 {
		t.Errorf("http = %d", w.Code)
	}
	env := decode(t, w.Body.String())
	if env.Code != 9999 {
		t.Errorf("code = %d", env.Code)
	}
}
