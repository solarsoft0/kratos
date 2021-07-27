package sms_test

import (
	"context"
	"github.com/ory/kratos/driver/config"
	"github.com/ory/kratos/internal"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_notificationClientImpl_Send(t *testing.T) {
	var serverInvoked bool
	conf, reg := internal.NewFastRegistryWithMocks(t)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serverInvoked = true
	}))
	t.Cleanup(ts.Close)
	conf.MustSet(config.SmsSenderUrl, ts.URL)

	tests := []struct {
		name    string
		phone   string
		code    string
		wantErr bool
	}{
		{
			"send code",
			"1234567",
			"0000",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			serverInvoked = false
			if err := reg.SmsNotificationClient().Send(context.Background(), tt.phone, tt.code); (err != nil) != tt.wantErr {
				t.Errorf("Send() error = %v, wantErr %v", err, tt.wantErr)
			}
			assert.True(t, serverInvoked)
		})
	}
}
