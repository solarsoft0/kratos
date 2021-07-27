package sms

// submitSelfServiceLoginFlowWithPasswordMethod is used to decode the login form payload.
//
// swagger:model submitSelfServiceLoginFlowWithSmsMethod
type submitSelfServiceLoginFlowWithSmsMethod struct {
	// Method should be set to "sms" when logging in using the sms strategy.
	Method string `json:"method"`

	// Sending the anti-csrf token is only required for browser login flows.
	CSRFToken string `json:"csrf_token"`

	// The user's phone number.
	Phone string `json:"phone"`

	// SMS one-time code.
	Code string `json:"code"`
}
