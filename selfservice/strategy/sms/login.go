package sms

import (
	"fmt"
	"github.com/ory/kratos/driver/config"
	"github.com/ory/kratos/identity"
	"github.com/ory/kratos/schema"
	"github.com/ory/kratos/selfservice/flow"
	"github.com/ory/kratos/selfservice/flow/login"
	"github.com/ory/kratos/session"
	"github.com/ory/kratos/text"
	"github.com/ory/kratos/ui/node"
	"github.com/ory/x/decoderx"
	"github.com/ory/x/sqlcon"
	"github.com/pkg/errors"
	"net/http"
)

func (s *Strategy) handleLoginError(w http.ResponseWriter, r *http.Request, f *login.Flow,
	payload *submitSelfServiceLoginFlowWithSmsMethod, err error) error {
	if f != nil {
		f.UI.Nodes.ResetNodes("code")
		f.UI.Nodes.SetValueAttribute("phone", payload.Phone)
		if f.Type == flow.TypeBrowser {
			f.UI.SetCSRF(s.d.GenerateCSRFToken(r))
		}
	}

	return err
}

func (s *Strategy) Login(w http.ResponseWriter, r *http.Request, f *login.Flow, ss *session.Session) (*identity.Identity, error) {
	if err := flow.MethodEnabledAndAllowedFromRequest(r, s.ID().String(), s.d); err != nil {
		return nil, err
	}

	var p submitSelfServiceLoginFlowWithSmsMethod
	if err := s.hd.Decode(r, &p,
		decoderx.HTTPDecoderSetValidatePayloads(true),
		decoderx.MustHTTPRawJSONSchemaCompiler(loginSchema),
		decoderx.HTTPDecoderJSONFollowsFormFormat()); err != nil {
		return nil, s.handleLoginError(w, r, f, &p, err)
	}

	if p.Code == "" {
		err := s.d.SmsAuthenticationService().SendCode(r.Context(), f, p.Phone)
		if err != nil {
			return nil, s.handleLoginError(w, r, f, &p, err)
		}
		f.UI.Nodes.Upsert(node.NewInputField("code", "", node.SmsGroup, node.InputAttributeTypeText))
		return nil, s.handleLoginError(w, r, f, &p, NewSmsCodeSentError())
	} else {
		err := s.d.SmsAuthenticationService().VerifyCode(r.Context(), f, p.Code)
		if err != nil {
			return nil, s.handleLoginError(w, r, f, &p, err)
		}
	}

	i, _, err := s.d.PrivilegedIdentityPool().FindByCredentialsIdentifier(r.Context(), s.ID(), p.Phone)
	if err != nil {

		if !errors.Is(err, sqlcon.ErrNoRows) {
			return nil, err
		}

		i = identity.NewIdentity(config.DefaultIdentityTraitsSchemaID)
		i.Traits = identity.Traits(fmt.Sprintf("{\"phone\": \"%s\"}", p.Phone))

		if err := s.d.IdentityValidator().Validate(r.Context(), i); err != nil {
			return nil, err
		} else if err := s.d.IdentityManager().Create(r.Context(), i); err != nil {
			if errors.Is(err, sqlcon.ErrUniqueViolation) {
				return nil, schema.NewDuplicateCredentialsError()
			}
			return nil, err
		}

		s.d.Audit().
			WithRequest(r).
			WithField("identity_id", i.ID).
			Info("A new identity has registered using self-service login auto-provisioning.")
	}

	return i, nil
}

func (s *Strategy) PopulateLoginMethod(r *http.Request, requestedAAL identity.AuthenticatorAssuranceLevel, sr *login.Flow) error {
	// This strategy can only solve AAL1
	if requestedAAL > identity.AuthenticatorAssuranceLevel1 {
		return nil
	}

	// This block adds the identifier (i.e. phone) to the method when the request is forced - as a hint for the user.
	var identifier string
	if !sr.IsForced() {
		// do nothing
	} else if sess, err := s.d.SessionManager().FetchFromRequest(r.Context(), r); err != nil {
		// do nothing
	} else if id, err := s.d.PrivilegedIdentityPool().GetIdentityConfidential(r.Context(), sess.IdentityID); err != nil {
		// do nothing
	} else if creds, ok := id.GetCredentials(s.ID()); !ok {
		// do nothing
	} else if len(creds.Identifiers) == 0 {
		// do nothing
	} else {
		identifier = creds.Identifiers[0]
	}

	sr.UI.SetCSRF(s.d.GenerateCSRFToken(r))
	sr.UI.SetNode(node.NewInputField("phone", identifier, node.SmsGroup,
		node.InputAttributeTypePhone, node.WithRequiredInputAttribute).WithMetaLabel(text.NewInfoNodeLabelID()))
	//sr.UI.SetNode(node.NewInputField("code", nil, node.SmsGroup, node.InputAttributeTypeText))
	sr.UI.GetNodes().Append(node.NewInputField("method", "sms", node.SmsGroup,
		node.InputAttributeTypeSubmit).WithMetaLabel(text.NewInfoLogin()))

	return nil
}
