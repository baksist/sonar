package server

import (
	"testing"

	"github.com/nt0xa/sonar/pkg/smtpx"
)

func TestShouldIgnoreSMTPEvent(t *testing.T) {
	matchingEvent := &smtpx.Event{
		Data:  &smtpx.Data{MailFrom: ignoreSMTPHotfixMailFrom},
		Email: smtpx.Email{Subject: ignoreSMTPHotfixSubject},
	}

	tests := []struct {
		name  string
		env   string
		event *smtpx.Event
		want  bool
	}{
		{
			name: "ignore disabled",
			env:  "0",
			event: &smtpx.Event{
				Data:  &smtpx.Data{MailFrom: ignoreSMTPHotfixMailFrom},
				Email: smtpx.Email{Subject: ignoreSMTPHotfixSubject},
			},
			want: false,
		},
		{
			name:  "ignore enabled and event matches",
			env:   "1",
			event: matchingEvent,
			want:  true,
		},
		{
			name: "ignore enabled but sender mismatches",
			env:  "1",
			event: &smtpx.Event{
				Data:  &smtpx.Data{MailFrom: "other@example.com"},
				Email: smtpx.Email{Subject: ignoreSMTPHotfixSubject},
			},
			want: false,
		},
		{
			name: "ignore enabled but subject mismatches",
			env:  "1",
			event: &smtpx.Event{
				Data:  &smtpx.Data{MailFrom: ignoreSMTPHotfixMailFrom},
				Email: smtpx.Email{Subject: "Other Subject"},
			},
			want: false,
		},
		{
			name:  "ignore enabled but event missing data",
			env:   "1",
			event: &smtpx.Event{},
			want:  false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(ignoreSMTPHotfixEnvVar, tc.env)

			if got := shouldIgnoreSMTPEvent(tc.event); got != tc.want {
				t.Fatalf("shouldIgnoreSMTPEvent() = %v, want %v", got, tc.want)
			}
		})
	}
}
