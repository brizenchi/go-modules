package smtp

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/brizenchi/go-modules/modules/email/domain"
)

func TestConfigValidate(t *testing.T) {
	cases := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"missing host", Config{Port: 25, Sender: domain.Address{Email: "a@b"}}, true},
		{"missing port", Config{Host: "h", Sender: domain.Address{Email: "a@b"}}, true},
		{"missing sender", Config{Host: "h", Port: 25}, true},
		{"ok", Config{Host: "h", Port: 25, Sender: domain.Address{Email: "a@b"}}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cfg.Validate()
			if (err != nil) != tc.wantErr {
				t.Fatalf("Validate err = %v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestSend_TemplateRefRejected(t *testing.T) {
	s, err := New(Config{Host: "h", Port: 25, Sender: domain.Address{Email: "from@x"}})
	if err != nil {
		t.Fatal(err)
	}
	_, err = s.Send(context.Background(), &domain.Message{
		To:          []domain.Address{{Email: "a@b"}},
		Subject:     "x",
		HTMLBody:    "<p/>",
		TemplateRef: "welcome",
	})
	if !errors.Is(err, domain.ErrSendFailed) {
		t.Fatalf("want ErrSendFailed, got %v", err)
	}
}

func TestBuildMIME_TextOnly(t *testing.T) {
	from := domain.Address{Name: "Acme", Email: "no-reply@acme.test"}
	body, err := buildMIME(from, &domain.Message{
		To:       []domain.Address{{Email: "bob@x.test"}},
		Subject:  "hi",
		TextBody: "plain body",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	for _, want := range []string{
		"From: \"Acme\" <no-reply@acme.test>",
		"To: bob@x.test",
		"Subject: hi",
		"MIME-Version: 1.0",
		"Content-Type: text/plain; charset=UTF-8",
		"plain body",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildMIME_HTMLOnly(t *testing.T) {
	body, err := buildMIME(domain.Address{Email: "f@x"}, &domain.Message{
		To:       []domain.Address{{Email: "a@b"}},
		Subject:  "hi",
		HTMLBody: "<p>hi</p>",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	if !strings.Contains(got, "Content-Type: text/html; charset=UTF-8") {
		t.Errorf("expected html content-type:\n%s", got)
	}
	if !strings.Contains(got, "<p>hi</p>") {
		t.Errorf("missing html body:\n%s", got)
	}
}

func TestBuildMIME_Multipart(t *testing.T) {
	body, err := buildMIME(domain.Address{Email: "f@x"}, &domain.Message{
		To:       []domain.Address{{Email: "a@b"}},
		Subject:  "hi",
		HTMLBody: "<p>hi</p>",
		TextBody: "plain",
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	if !strings.Contains(got, "multipart/alternative; boundary=") {
		t.Errorf("expected multipart:\n%s", got)
	}
	if !strings.Contains(got, "<p>hi</p>") || !strings.Contains(got, "plain") {
		t.Errorf("both parts should be present:\n%s", got)
	}
}

func TestBuildMIME_HeadersAndReplyTo(t *testing.T) {
	body, err := buildMIME(domain.Address{Email: "f@x"}, &domain.Message{
		To:       []domain.Address{{Email: "a@b"}},
		Cc:       []domain.Address{{Email: "c@b"}},
		ReplyTo:  &domain.Address{Email: "r@x"},
		Subject:  "hi",
		HTMLBody: "<p/>",
		Headers:  []domain.Header{{Name: "X-Tag", Value: "abc"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	got := string(body)
	for _, want := range []string{"Cc: c@b", "Reply-To: r@x", "X-Tag: abc"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestBuildMIME_AttachmentsRejected(t *testing.T) {
	_, err := buildMIME(domain.Address{Email: "f@x"}, &domain.Message{
		To:          []domain.Address{{Email: "a@b"}},
		Subject:     "hi",
		HTMLBody:    "<p/>",
		Attachments: []domain.Attachment{{Name: "f.pdf"}},
	})
	if err == nil {
		t.Fatal("expected attachments rejection")
	}
}

func TestFormatAddrAndJoin(t *testing.T) {
	if got := formatAddr(domain.Address{Email: "a@b"}); got != "a@b" {
		t.Errorf("plain: %q", got)
	}
	if got := formatAddr(domain.Address{Name: "Bob", Email: "b@x"}); got != `"Bob" <b@x>` {
		t.Errorf("named: %q", got)
	}
	if got := joinAddrs([]domain.Address{{Email: "a@b"}, {Email: "c@d"}}); got != "a@b, c@d" {
		t.Errorf("join: %q", got)
	}
}
