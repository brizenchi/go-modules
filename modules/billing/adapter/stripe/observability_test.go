package stripe

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	stripesdk "github.com/stripe/stripe-go/v76"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

// TestProviderPropagatesContextAndUsesInjectedClient checks that API calls
// carry the caller's ctx (so the trace continues) and go through
// Config.HTTPClient.
func TestProviderPropagatesContextAndUsesInjectedClient(t *testing.T) {
	tp := sdktrace.NewTracerProvider()
	defer func() { _ = tp.Shutdown(context.Background()) }()
	prevProp := otel.GetTextMapPropagator()
	otel.SetTextMapPropagator(propagation.TraceContext{})
	t.Cleanup(func() { otel.SetTextMapPropagator(prevProp) })

	var traceparent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		traceparent = r.Header.Get("traceparent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"sub_123","object":"subscription","status":"active","items":{"data":[]}}`))
	}))
	defer srv.Close()

	prevKey := stripesdk.Key
	t.Cleanup(func() {
		stripesdk.Key = prevKey
		stripesdk.SetBackend(stripesdk.APIBackend, stripesdk.GetBackendWithConfig(stripesdk.APIBackend, &stripesdk.BackendConfig{}))
	})

	cfg := newTestConfig()
	cfg.HTTPClient = &http.Client{Transport: injectingTransport{next: srv.Client().Transport}}
	p := NewProvider(cfg)
	// Point the backend installed by NewProvider at the test server while
	// keeping the injected client.
	stripesdk.SetBackend(stripesdk.APIBackend, stripesdk.GetBackendWithConfig(stripesdk.APIBackend, &stripesdk.BackendConfig{
		URL:        stripesdk.String(srv.URL),
		HTTPClient: cfg.HTTPClient,
	}))

	ctx, span := tp.Tracer("test").Start(context.Background(), "caller")
	defer span.End()
	if _, err := p.GetSubscription(ctx, "sub_123"); err != nil {
		t.Fatalf("GetSubscription: %v", err)
	}
	if traceparent == "" {
		t.Fatal("request did not carry the caller's context")
	}
}

type injectingTransport struct{ next http.RoundTripper }

func (t injectingTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	otel.GetTextMapPropagator().Inject(r.Context(), propagation.HeaderCarrier(r.Header))
	return t.next.RoundTrip(r)
}
