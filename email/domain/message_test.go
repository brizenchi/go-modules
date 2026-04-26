package domain

import (
	"errors"
	"testing"
)

func TestMessage_Validate(t *testing.T) {
	cases := []struct {
		name    string
		msg     Message
		wantErr error
	}{
		{
			name:    "no recipients",
			msg:     Message{Subject: "hi", TextBody: "x"},
			wantErr: ErrInvalidRecipient,
		},
		{
			name:    "empty recipient email",
			msg:     Message{To: []Address{{Email: ""}}, Subject: "hi", TextBody: "x"},
			wantErr: ErrInvalidRecipient,
		},
		{
			name:    "missing subject and template",
			msg:     Message{To: []Address{{Email: "a@b"}}, TextBody: "x"},
			wantErr: ErrEmptyContent,
		},
		{
			name:    "missing both bodies",
			msg:     Message{To: []Address{{Email: "a@b"}}, Subject: "s"},
			wantErr: ErrEmptyContent,
		},
		{
			name: "ok with text body",
			msg:  Message{To: []Address{{Email: "a@b"}}, Subject: "s", TextBody: "x"},
		},
		{
			name: "ok with html body",
			msg:  Message{To: []Address{{Email: "a@b"}}, Subject: "s", HTMLBody: "<p>x</p>"},
		},
		{
			name: "ok with template_ref skipping subject/body checks",
			msg:  Message{To: []Address{{Email: "a@b"}}, TemplateRef: "3"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.Validate()
			if tc.wantErr == nil {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				return
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("error = %v, want errors.Is %v", err, tc.wantErr)
			}
		})
	}
}
