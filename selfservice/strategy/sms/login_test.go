package sms_test

import (
	"context"
	"fmt"
	kratos "github.com/ory/kratos-client-go"
	"github.com/ory/kratos/driver/config"
	"github.com/ory/kratos/identity"
	"github.com/ory/kratos/internal"
	"github.com/ory/kratos/internal/testhelpers"
	"github.com/ory/kratos/selfservice/flow/login"
	"github.com/ory/kratos/selfservice/strategy/sms"
	"github.com/ory/kratos/session"
	"github.com/ory/kratos/x"
	"github.com/ory/x/errorsx"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/tidwall/gjson"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"testing"
)

func newReturnTs(t *testing.T, reg interface {
	session.ManagementProvider
	x.WriterProvider
	config.Provider
}) *httptest.Server {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sess, err := reg.SessionManager().FetchFromRequest(r.Context(), r)
		require.NoError(t, err)
		reg.Writer().Write(w, r, sess)
	}))
	t.Cleanup(ts.Close)
	reg.Config(context.Background()).MustSet(config.ViperKeySelfServiceBrowserDefaultReturnTo, ts.URL+"/return-ts")
	return ts
}

func checkFormContent(t *testing.T, body []byte, requiredFields ...string) {
	fieldNameSet(t, body, requiredFields)
	formMethodIsPOST(t, body)
}

// fieldNameSet checks if the fields have the right "name" set.
func fieldNameSet(t *testing.T, body []byte, fields []string) {
	for _, f := range fields {
		assert.Equal(t, f, gjson.GetBytes(body, fmt.Sprintf("ui.nodes.#(attributes.name==%s).attributes.name", f)).String(), "%s", body)
	}
}

func formMethodIsPOST(t *testing.T, body []byte) {
	assert.Equal(t, "POST", gjson.GetBytes(body, "ui.method").String())
}

func TestStrategy_Login(t *testing.T) {
	conf, reg := internal.NewFastRegistryWithMocks(t)
	reg.WithRandomCodeGenerator(&randomCodeGeneratorStub{code: "0000"})
	conf.MustSet(config.ViperKeySelfServiceStrategyConfig+"."+string(identity.CredentialsTypeSMS)+".enabled", true)
	conf.MustSet(config.ViperKeySelfServiceStrategyConfig+"."+string(identity.CredentialsTypePassword)+".enabled", false)
	conf.MustSet(config.ViperKeyDefaultIdentitySchemaURL, "file://./stub/registration.schema.json")
	conf.MustSet(config.ViperKeyCourierSMTPURL, "smtp://foo@bar@dev.null/")
	conf.MustSet(config.ViperKeyCourierSMSHost, "http://foo.url")
	publicTS, _ := testhelpers.NewKratosServer(t, reg)
	redirTS := newReturnTs(t, reg)

	uiTS := testhelpers.NewLoginUIFlowEchoServer(t, reg)

	conf.MustSet(config.ViperKeySelfServiceLoginUI, uiTS.URL+"/login-ts")

	var smsSenderResponseStatus int
	var _, realSmsService = os.LookupEnv("TEST_REAL_SMS_SERVICE")
	if !realSmsService {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(smsSenderResponseStatus)
			if _, err := fmt.Fprintf(w, "{\"Response\": \"Test response\"}"); err != nil {
				log.Fatal(err)
			}
		}))
		t.Cleanup(ts.Close)
		conf.MustSet(config.SmsSenderUrl, ts.URL)
	} else {
		conf.MustSet(config.SmsSenderUrl, "http://127.0.0.1:8083/api/user/sms")
	}

	ensureFieldsExist := func(t *testing.T, body []byte) {
		checkFormContent(t, body, "phone", "csrf_token")
	}

	var expectValidationError = func(t *testing.T, isAPI, forced, isSPA bool, values func(url.Values)) string {
		return testhelpers.SubmitLoginForm(t, isAPI, nil, publicTS, values,
			isSPA, forced,
			testhelpers.ExpectStatusCode(isAPI || isSPA, http.StatusBadRequest, http.StatusOK),
			testhelpers.ExpectURL(isAPI || isSPA, publicTS.URL+login.RouteSubmitFlow, conf.SelfServiceFlowLoginUI().String()),
		)
	}

	t.Run("should return an error because no phone is set", func(t *testing.T) {
		smsSenderResponseStatus = http.StatusOK
		var check = func(t *testing.T, body string) {
			assert.NotEmpty(t, gjson.Get(body, "id").String(), "%s", body)
			assert.Contains(t, gjson.Get(body, "ui.action").String(), publicTS.URL+login.RouteSubmitFlow, "%s", body)

			ensureFieldsExist(t, []byte(body))
			assert.Equal(t, "Property phone is missing.", gjson.Get(body, "ui.nodes.#(attributes.name==phone).messages.0.text").String(), "%s", body)
			assert.Len(t, gjson.Get(body, "ui.nodes").Array(), 3)

			// The SMS code value should not be returned!
			assert.Empty(t, gjson.Get(body, "ui.nodes.#(attributes.name==code).attributes.value").String())
		}

		var values = func(v url.Values) {
			v.Del("phone")
		}

		t.Run("type=api", func(t *testing.T) {
			check(t, expectValidationError(t, true, false, false, values))
		})
	})

	t.Run("should fail since bad request to the sms service", func(t *testing.T) {
		smsSenderResponseStatus = http.StatusBadRequest

		t.Run("type=api", func(t *testing.T) {
			f := testhelpers.InitializeLoginFlow(t, true, nil, publicTS, false, false)

			var values = func(v url.Values) {
				v.Set("phone", "bad_phone_number")
			}

			testhelpers.SubmitLoginFormWithFlow(t, true, nil, values,
				false, http.StatusInternalServerError, publicTS.URL+login.RouteSubmitFlow, f)
		})

	})

	t.Run("should pass with real request", func(t *testing.T) {
		smsSenderResponseStatus = http.StatusOK
		identifier := "+11111111111"

		t.Run("type=api", func(t *testing.T) {

			f := testhelpers.InitializeLoginFlow(t, true, nil, publicTS, false, false)

			assert.Empty(t, getLoginNode(f, "code"))
			assert.NotEmpty(t, getLoginNode(f, "phone"))

			var values = func(v url.Values) {
				v.Set("phone", identifier)
			}
			body := testhelpers.SubmitLoginFormWithFlow(t, true, nil, values,
				false, http.StatusOK, publicTS.URL+login.RouteSubmitFlow, f)

			st := gjson.Get(body, "session_token").String()
			assert.Empty(t, st, "Response body: %s", body) //No session token as we have not presented the SMS code yet

			values = func(v url.Values) {
				v.Set("phone", identifier)
				v.Set("code", "0000")
			}

			body = testhelpers.SubmitLoginFormWithFlow(t, true, nil, values,
				false, http.StatusOK, publicTS.URL+login.RouteSubmitFlow, f)

			assert.Equal(t, identifier, gjson.Get(body, "session.identity.traits.phone").String(),
				"%s", body)
			st = gjson.Get(body, "session_token").String()
			assert.NotEmpty(t, st, "%s", body)
		})

		t.Run("type=browser", func(t *testing.T) {

			f := testhelpers.InitializeLoginFlow(t, false, nil, publicTS, false, false)

			assert.Empty(t, getLoginNode(f, "code"))
			assert.NotEmpty(t, getLoginNode(f, "phone"))

			var values = func(v url.Values) {
				v.Set("phone", identifier)
			}
			body := testhelpers.SubmitLoginFormWithFlow(t, false, nil, values,
				false, http.StatusOK, conf.SelfServiceFlowLoginUI().String(), f)

			assert.Equal(t,
				errorsx.Cause(sms.NewSmsCodeSentError()).(sms.CodeSentError).ValidationError.Messages[0].Text,
				gjson.Get(body, "ui.messages.0.text").String(),
				"%s", body,
			)
			assert.NotEmpty(t, gjson.Get(body, "ui.nodes.#(attributes.name==code)"), "%s", body)
			assert.Empty(t, gjson.Get(body, "ui.nodes.#(attributes.name==code).attirbutes.value"), "%s", body)

			values = func(v url.Values) {
				v.Set("phone", identifier)
				v.Set("code", "0000")
			}

			body = testhelpers.SubmitLoginFormWithFlow(t, false, nil, values,
				false, http.StatusOK, redirTS.URL, f)

			assert.Equal(t, identifier, gjson.Get(body, "identity.traits.phone").String(),
				"%s", body)

		})
	})
}

func getLoginNode(f *kratos.SelfServiceLoginFlow, nodeName string) *kratos.UiNode {
	for _, n := range f.Ui.Nodes {
		if n.Attributes.UiNodeInputAttributes.Name == nodeName {
			return &n
		}
	}
	return nil
}
