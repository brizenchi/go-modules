package s3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/foundation/ossx"
)

func TestPutOmitsPrivateACLForS3AndR2Compatibility(t *testing.T) {
	for _, tc := range []struct {
		name string
		acl  ossx.ACL
		want string
	}{{"default", "", ""}, {"private", ossx.ACLPrivate, ""}, {"explicit public", ossx.ACLPublicRead, "public-read"}} {
		t.Run(tc.name, func(t *testing.T) {
			headers := make(chan string, 1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				headers <- r.Header.Get("X-Amz-Acl")
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()
			bucket, err := New(context.Background(), Config{Bucket: "private-bucket", Region: "auto", Endpoint: server.URL, UsePathStyle: true, AccessKeyID: "test-access", SecretAccessKey: "test-secret"})
			if err != nil {
				t.Fatal(err)
			}
			if err := bucket.Put(context.Background(), "owner/image.png", strings.NewReader("image"), 5, ossx.PutOptions{ContentType: "image/png", ACL: tc.acl}); err != nil {
				t.Fatal(err)
			}
			if actual := <-headers; actual != tc.want {
				t.Fatalf("ACL=%q want%q", actual, tc.want)
			}
		})
	}
}
