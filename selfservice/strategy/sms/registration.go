package sms

import (
	"encoding/json"
	"fmt"
	"github.com/ory/kratos/identity"
	"github.com/ory/kratos/selfservice/flow"
	"github.com/ory/kratos/selfservice/flow/registration"
	"github.com/ory/kratos/ui/container"
	"github.com/ory/kratos/ui/node"
	"github.com/ory/kratos/x"
	"github.com/ory/x/decoderx"
	"github.com/pkg/errors"
	"github.com/tidwall/sjson"
	"net/http"
)

// SubmitSelfServiceRegistrationFlowWithSmsMethodBody is used to decode the registration form payload
// when using the sms method.
//
// swagger:model submitSelfServiceRegistrationFlowWithSmsMethodBody
type SubmitSelfServiceRegistrationFlowWithSmsMethodBody struct {
	// Code from the sms
	//
	// required: false
	Code string `json:"code"`

	// The identity's traits
	//
	// required: true
	Traits json.RawMessage `json:"traits"`

	// The CSRF Token
	CSRFToken string `json:"csrf_token"`

	// Method to use
	//
	// This field must be set to `sms` when using the sms method.
	//
	// required: true
	Method string `json:"method"`
}

func (s *Strategy) RegisterRegistrationRoutes(_ *x.RouterPublic) {
}

func (s *Strategy) handleRegistrationError(_ http.ResponseWriter, r *http.Request, f *registration.Flow,
	p *SubmitSelfServiceRegistrationFlowWithSmsMethodBody, err error) error {
	if f != nil {
		if p != nil {
			for _, n := range container.NewFromJSON("", node.SmsGroup, p.Traits, "traits").Nodes {
				// we only set the value and not the whole field because we want to keep types from the initial form generation
				f.UI.Nodes.SetValueAttribute(n.ID(), n.Attributes.GetValue())
			}
		}

		if f.Type == flow.TypeBrowser {
			f.UI.SetCSRF(s.d.GenerateCSRFToken(r))
		}
	}

	return err
}

func (s *Strategy) decode(p *SubmitSelfServiceRegistrationFlowWithSmsMethodBody, r *http.Request) error {
	raw, err := sjson.SetBytes(registrationSchema,
		"properties.traits.$ref", s.d.Config(r.Context()).DefaultIdentityTraitsSchemaURL().String()+"#/properties/traits")
	if err != nil {
		return errors.WithStack(err)
	}

	compiler, err := decoderx.HTTPRawJSONSchemaCompiler(raw)
	if err != nil {
		return errors.WithStack(err)
	}

	return s.hd.Decode(r, p, compiler, decoderx.HTTPDecoderSetValidatePayloads(true), decoderx.HTTPDecoderJSONFollowsFormFormat())
}

func (s *Strategy) Register(w http.ResponseWriter, r *http.Request, f *registration.Flow, i *identity.Identity) error {
	if err := flow.MethodEnabledAndAllowedFromRequest(r, s.ID().String(), s.d); err != nil {
		return err
	}

	var p SubmitSelfServiceRegistrationFlowWithSmsMethodBody
	if err := s.decode(&p, r); err != nil {
		return s.handleRegistrationError(w, r, f, &p, err)
	}

	if err := flow.EnsureCSRF(s.d, r, f.Type, s.d.Config(r.Context()).DisableAPIFlowEnforcement(), s.d.GenerateCSRFToken, p.CSRFToken); err != nil {
		return s.handleRegistrationError(w, r, f, &p, err)
	}

	if len(p.Traits) == 0 {
		p.Traits = json.RawMessage("{}")
	}

	i.Traits = identity.Traits(p.Traits)

	if p.Code == "" {
		if err := s.d.IdentityValidator().Validate(r.Context(), i); err != nil {
			return err
		}
		credentials, found := i.GetCredentials(identity.CredentialsTypeSMS)
		if !found {
			return s.handleRegistrationError(w, r, f, &p, fmt.Errorf("Credentials not found"))
		}
		if len(credentials.Identifiers) != 1 {
			return s.handleRegistrationError(w, r, f, &p,
				fmt.Errorf("Credentials identifiers missing or more than one: %v", credentials.Identifiers))
		}
		err := s.d.SmsAuthenticationService().SendCode(r.Context(), f, credentials.Identifiers[0])
		if err != nil {
			return s.handleRegistrationError(w, r, f, &p, err)
		}
		f.UI.Nodes.Upsert(node.NewInputField("code", "", node.SmsGroup, node.InputAttributeTypeText))
		return s.handleRegistrationError(w, r, f, &p, NewSmsCodeSentError())
	} else {
		err := s.d.SmsAuthenticationService().VerifyCode(r.Context(), f, p.Code)
		if err != nil {
			return s.handleRegistrationError(w, r, f, &p, err)
		}
	}

	return nil
}

func (s *Strategy) PopulateRegistrationMethod(r *http.Request, f *registration.Flow) error {
	return nil
}
