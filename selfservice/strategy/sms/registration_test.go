package sms_test

import (
	"context"
	"encoding/json"
	kratos "github.com/ory/kratos-client-go"
	"github.com/ory/kratos/courier/template"
	"github.com/ory/kratos/driver/config"
	"github.com/ory/kratos/identity"
	"github.com/ory/kratos/internal"
	"github.com/ory/kratos/internal/testhelpers"
	"github.com/ory/kratos/selfservice/flow/registration"
	"github.com/ory/kratos/x"
	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
	"net/http"
	"net/url"
	"testing"
)

func TestRegistration(t *testing.T) {
	t.Run("case=registration", func(t *testing.T) {
		conf, reg := internal.NewFastRegistryWithMocks(t)

		router := x.NewRouterPublic()
		admin := x.NewRouterAdmin()
		conf.MustSet(config.ViperKeySelfServiceStrategyConfig+"."+string(identity.CredentialsTypeSMS), map[string]interface{}{"enabled": true})

		publicTS, _ := testhelpers.NewKratosServerWithRouters(t, reg, router, admin)
		//errTS := testhelpers.NewErrorTestServer(t, reg)
		//uiTS := testhelpers.NewRegistrationUIFlowEchoServer(t, reg)
		redirTS := testhelpers.NewRedirSessionEchoTS(t, reg)

		// Overwrite these two to ensure that they run
		conf.MustSet(config.ViperKeySelfServiceBrowserDefaultReturnTo, redirTS.URL+"/default-return-to")
		conf.MustSet(config.ViperKeySelfServiceRegistrationAfter+"."+config.DefaultBrowserReturnURL, redirTS.URL+"/registration-return-ts")
		conf.MustSet(config.ViperKeyDefaultIdentitySchemaURL, "file://./stub/registration.schema.json")
		conf.MustSet(config.ViperKeyCourierSMSHost, "http://foo.url")

		var expectSuccessfulLogin = func(
			t *testing.T, isAPI, isSPA bool, hc *http.Client,
			expectReturnTo string,
			identifier string,
		) string {
			if hc == nil {
				if isAPI {
					hc = new(http.Client)
				} else {
					hc = testhelpers.NewClientWithCookies(t)
				}
			}

			_, err := reg.CourierPersister().NextMessages(context.Background(), 10)
			assert.Error(t, err, "Courier queue should be empty.")

			f := testhelpers.InitializeRegistrationFlow(t, isAPI, hc, publicTS, isSPA)

			assert.Empty(t, getRegistrationNode(f, "code"))
			assert.NotEmpty(t, getRegistrationNode(f, "traits.phone"))

			var values = func(v url.Values) {
				v.Set("method", "sms")
				v.Set("traits.phone", identifier)
			}
			body := testhelpers.SubmitRegistrationFormWithFlow(t, isAPI, hc, values,
				isSPA, http.StatusOK, expectReturnTo, f)

			messages, err := reg.CourierPersister().NextMessages(context.Background(), 10)
			assert.NoError(t, err, "Courier queue should not be empty.")
			assert.Equal(t, 1, len(messages))
			var smsModel template.SmsLoginModel
			err = json.Unmarshal(messages[0].TemplateData, &smsModel)
			assert.NoError(t, err)

			st := gjson.Get(body, "session_token").String()
			assert.Empty(t, st, "Response body: %s", body) //No session token as we have not presented the SMS code yet

			values = func(v url.Values) {
				v.Set("method", "sms")
				v.Set("traits.phone", identifier)
				v.Set("code", smsModel.Code)
			}

			body = testhelpers.SubmitRegistrationFormWithFlow(t, isAPI, hc, values,
				isSPA, http.StatusOK, expectReturnTo, f)

			assert.Equal(t, identifier, gjson.Get(body, "session.identity.traits.phone").String(),
				"%s", body)
			st = gjson.Get(body, "session_token").String()
			assert.NotEmpty(t, st, "%s", body)

			return body
		}

		t.Run("case=should pass and set up a session", func(t *testing.T) {
			conf.MustSet(config.ViperKeyDefaultIdentitySchemaURL, "file://./stub/registration.schema.json")
			conf.MustSet(config.HookStrategyKey(config.ViperKeySelfServiceRegistrationAfter, identity.CredentialsTypeSMS.String()), []config.SelfServiceHook{{Name: "session"}})
			t.Cleanup(func() {
				conf.MustSet(config.HookStrategyKey(config.ViperKeySelfServiceRegistrationAfter, identity.CredentialsTypeSMS.String()), nil)
			})

			identifier := "+11111111111"

			t.Run("type=api", func(t *testing.T) {
				expectSuccessfulLogin(t, true, false, nil,
					publicTS.URL+registration.RouteSubmitFlow, identifier)
			})

			//t.Run("type=spa", func(t *testing.T) {
			//	hc := testhelpers.NewClientWithCookies(t)
			//	body := expectSuccessfulLogin(t, false, true, hc, func(v url.Values) {
			//		v.Set("traits.username", "registration-identifier-8-spa")
			//		v.Set("password", x.NewUUID().String())
			//		v.Set("traits.foobar", "bar")
			//	})
			//	assert.Equal(t, `registration-identifier-8-spa`, gjson.Get(body, "identity.traits.username").String(), "%s", body)
			//	assert.Empty(t, gjson.Get(body, "session_token").String(), "%s", body)
			//	assert.NotEmpty(t, gjson.Get(body, "session.id").String(), "%s", body)
			//})
			//
			//t.Run("type=browser", func(t *testing.T) {
			//	body := expectSuccessfulLogin(t, false, false, nil, func(v url.Values) {
			//		v.Set("traits.username", "registration-identifier-8-browser")
			//		v.Set("password", x.NewUUID().String())
			//		v.Set("traits.foobar", "bar")
			//	})
			//	assert.Equal(t, `registration-identifier-8-browser`, gjson.Get(body, "identity.traits.username").String(), "%s", body)
			//})
		})
	})
}

func getRegistrationNode(f *kratos.SelfServiceRegistrationFlow, nodeName string) *kratos.UiNode {
	for _, n := range f.Ui.Nodes {
		if n.Attributes.UiNodeInputAttributes.Name == nodeName {
			return &n
		}
	}
	return nil
}
