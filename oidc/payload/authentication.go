/*
 * Copyright 2017 Kopano and its licensors
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License, version 3,
 * as published by the Free Software Foundation.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 *
 */

package payload

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"

	"stash.kopano.io/kc/konnect/oidc"
)

// AuthenticationRequest holds the incoming parameters and request data for
// the OpenID Connect 1.0 authorization endpoint as specified at
// http://openid.net/specs/openid-connect-core-1_0.html#AuthRequest and
// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthRequest
type AuthenticationRequest struct {
	providerMetadata *WellKnown

	RawScope        string         `schema:"scope"`
	Claims          *ClaimsRequest `schema:"claims"`
	RawResponseType string         `schema:"response_type"`
	ResponseMode    string         `schema:"response_mode"`
	ClientID        string         `schema:"client_id"`
	RawRedirectURI  string         `schema:"redirect_uri"`
	State           string         `schema:"state"`
	Nonce           string         `schema:"nonce"`
	RawPrompt       string         `schema:"prompt"`
	RawIDTokenHint  string         `schema:"id_token_hint"`
	RawMaxAge       string         `schema:"max_age"`

	RawRequest      string `schema:"request"`
	RawRequestURI   string `schema:"request_uri"`
	RawRegistration string `schema:"registration"`

	Scopes        map[string]bool `schema:"-"`
	ResponseTypes map[string]bool `schema:"-"`
	Prompts       map[string]bool `schema:"-"`
	RedirectURI   *url.URL        `schema:"-"`
	IDTokenHint   *jwt.Token      `schema:"-"`
	MaxAge        time.Duration   `schema:"-"`
	Request       *jwt.Token      `schema:"-"`

	UseFragment bool   `schema:"-"`
	Flow        string `schema:"-"`
}

// DecodeAuthenticationRequest returns a AuthenticationRequest holding the
// provided requests form data.
func DecodeAuthenticationRequest(req *http.Request, providerMetadata *WellKnown, keyFunc jwt.Keyfunc) (*AuthenticationRequest, error) {
	return NewAuthenticationRequest(req.Form, providerMetadata, keyFunc)
}

// NewAuthenticationRequest returns a AuthenticationRequest holding the
// provided url values.
func NewAuthenticationRequest(values url.Values, providerMetadata *WellKnown, keyFunc jwt.Keyfunc) (*AuthenticationRequest, error) {
	ar := &AuthenticationRequest{
		providerMetadata: providerMetadata,

		Scopes:        make(map[string]bool),
		ResponseTypes: make(map[string]bool),
		Prompts:       make(map[string]bool),
	}
	err := DecodeSchema(ar, values)
	if err != nil {
		return nil, fmt.Errorf("failed to decode authentication request: %v", err)
	}

	if ar.RawScope != "" {
		// Parse scope early, since the value is needed to handle the request
		// parameter properly.
		for _, scope := range strings.Split(ar.RawScope, " ") {
			ar.Scopes[scope] = true
		}
	}

	if ar.RawRequest != "" {
		parser := &jwt.Parser{}
		request, err := parser.ParseWithClaims(ar.RawRequest, &RequestObjectClaims{}, func(token *jwt.Token) (interface{}, error) {
			if keyFunc != nil {
				return keyFunc(token)
			}

			return nil, fmt.Errorf("Not validated")
		})
		if err != nil {
			return nil, ar.NewBadRequest(oidc.ErrorOAuth2InvalidRequest, err.Error())
		}

		if claims, ok := request.Claims.(*RequestObjectClaims); ok {
			err = ar.ApplyRequestObject(claims, request.Method)
			if err != nil {
				return nil, err
			}
		}

		ar.Request = request
	}

	ar.RedirectURI, _ = url.Parse(ar.RawRedirectURI)

	if ar.RawResponseType != "" {
		for _, rt := range strings.Split(ar.RawResponseType, " ") {
			ar.ResponseTypes[rt] = true
		}
	}
	if ar.RawPrompt != "" {
		for _, prompt := range strings.Split(ar.RawPrompt, " ") {
			ar.Prompts[prompt] = true
		}
	}

	switch ar.RawResponseType {
	case oidc.ResponseTypeCode:
		// Code flow.
		ar.Flow = oidc.FlowCode
		// breaks
	case oidc.ResponseTypeIDToken:
		// Implicit flow.
		fallthrough
	case oidc.ResponseTypeIDTokenToken:
		// Implicit flow with access token.
		ar.UseFragment = true
		ar.Flow = oidc.FlowImplicit
	case oidc.ResponseTypeCodeIDToken:
		// Hybrid flow.
		fallthrough
	case oidc.ResponseTypeCodeToken:
		// Hybgrid flow.
		fallthrough
	case oidc.ResponseTypeCodeIDTokenToken:
		// Hybrid flow.
		ar.UseFragment = true
		ar.Flow = oidc.FlowHybrid
	}

	switch ar.ResponseMode {
	case oidc.ResponseModeFragment:
		ar.UseFragment = true
		// breaks
	case oidc.ResponseModeQuery:
		ar.UseFragment = false
		// breaks
	}

	if ar.RawMaxAge != "" {
		maxAgeInt, err := strconv.ParseInt(ar.RawMaxAge, 10, 64)
		if err != nil {
			return nil, err
		}
		ar.MaxAge = time.Duration(maxAgeInt) * time.Second
	}

	return ar, nil
}

// ApplyRequestObject applies the provided request object claims to the
// associated authentication request data with validation as required.
func (ar *AuthenticationRequest) ApplyRequestObject(roc *RequestObjectClaims, method jwt.SigningMethod) error {
	// Basic consistency validation following spec at
	// https://openid.net/specs/openid-connect-core-1_0.html#SignedRequestObject
	if ok := ar.Scopes[oidc.ScopeOpenID]; !ok {
		return ar.NewBadRequest(oidc.ErrorOAuth2InvalidRequest, "openid scope required when using the request parameter")
	}
	if roc.RawScope != "" {
		ar.Scopes = make(map[string]bool)
		// Parse scope directly, since the accociated authentication request
		// has already parsed it when this is called.
		for _, scope := range strings.Split(roc.RawScope, " ") {
			ar.Scopes[scope] = true
		}
	}
	if roc.RawResponseType != "" {
		if roc.RawResponseType != ar.RawResponseType {
			return ar.NewBadRequest(oidc.ErrorOAuth2InvalidRequest, "request object response_type mismatch")
		}
	}
	if roc.ClientID != "" {
		if roc.ClientID != ar.ClientID {
			return ar.NewBadRequest(oidc.ErrorOAuth2InvalidRequest, "request object client_id mismatch")
		}
	}

	if method != jwt.SigningMethodNone {
		// Additional claim validation when signed. The spec says that iss and
		// aud SHOULD have defined values. So for now we do not enforce here.
	}

	// Apply rest of the provided request object values to the accociated
	// authentication request.
	if roc.Claims != nil {
		ar.Claims = roc.Claims
	}
	if roc.RawRedirectURI != "" {
		ar.RawRedirectURI = roc.RawRedirectURI
	}
	if roc.State != "" {
		ar.State = roc.State
	}
	if roc.Nonce != "" {
		ar.Nonce = roc.Nonce
	}
	if roc.RawPrompt != "" {
		ar.RawPrompt = roc.RawPrompt
	}
	if roc.RawIDTokenHint != "" {
		ar.RawIDTokenHint = roc.RawIDTokenHint
	}
	if roc.RawMaxAge != "" {
		ar.RawMaxAge = roc.RawMaxAge
	}
	if roc.RawRegistration != "" {
		ar.RawRegistration = roc.RawRegistration
	}

	return nil
}

// Validate validates the request data of the accociated authentication request.
func (ar *AuthenticationRequest) Validate(keyFunc jwt.Keyfunc) error {
	if _, ok := ar.Scopes[oidc.ScopeOpenID]; !ok {
		return ar.NewBadRequest(oidc.ErrorOAuth2InvalidRequest, "missing openid scope in request")
	}

	switch ar.RawResponseType {
	case oidc.ResponseTypeCode:
		// Code flow.
		// breaks
	case oidc.ResponseTypeCodeIDToken:
		// Hybgrid flow.
		// breaks
	case oidc.ResponseTypeCodeToken:
		// Hybgrid flow.
		// breaks
	case oidc.ResponseTypeCodeIDTokenToken:
		// Hybgrid flow.
		// breaks
	case oidc.ResponseTypeIDToken:
		// Implicit flow.
		fallthrough
	case oidc.ResponseTypeIDTokenToken:
		// Implicit flow with access token.
		if ar.Nonce == "" {
			return ar.NewError(oidc.ErrorOAuth2InvalidRequest, "nonce is required for implicit flow")
		}
	case oidc.ResponseTypeToken:
		// OAuth2 flow implicit grant.
		// breaks
	default:
		return ar.NewError(oidc.ErrorOAuth2UnsupportedResponseType, "")
	}

	if _, hasNonePrompt := ar.Prompts[oidc.PromptNone]; hasNonePrompt {
		if len(ar.Prompts) > 1 {
			// Cannot have other prompts if none is requested.
			return ar.NewError(oidc.ErrorOAuth2InvalidRequest, "cannot request other prompts together with none")
		}
	}

	if ar.ClientID == "" {
		return ar.NewBadRequest(oidc.ErrorOAuth2InvalidRequest, "missing client_id")
	}
	// TODO(longsleep): implement client_id white list.

	if ar.RedirectURI == nil || ar.RedirectURI.Host == "" || ar.RedirectURI.Scheme == "" {
		return ar.NewBadRequest(oidc.ErrorOAuth2InvalidRequest, "invalid or missing redirect_uri")
	}

	if ar.RawIDTokenHint != "" {
		parser := &jwt.Parser{
			SkipClaimsValidation: true,
		}
		idTokenHint, err := parser.ParseWithClaims(ar.RawIDTokenHint, &oidc.IDTokenClaims{}, func(token *jwt.Token) (interface{}, error) {
			if keyFunc != nil {
				return keyFunc(token)
			}

			return nil, fmt.Errorf("Not validated")
		})
		if err != nil {
			return ar.NewBadRequest(oidc.ErrorOAuth2InvalidRequest, err.Error())
		}
		ar.IDTokenHint = idTokenHint
	}

	// Offline access validation.
	// http://openid.net/specs/openid-connect-core-1_0.html#OfflineAccess
	if ok, _ := ar.Scopes[oidc.ScopeOfflineAccess]; ok {
		if _, withCodeResponseType := ar.ResponseTypes[oidc.ResponseTypeCode]; !withCodeResponseType {
			// Ignore the offline_access request unless the Client is using a
			// response_type value that would result in an Authorization Code
			// being returned.
			delete(ar.Scopes, oidc.ScopeOfflineAccess)
		}
	}

	if ar.RawRequestURI != "" {
		return ar.NewError(oidc.ErrorOIDCRequestURINotSupported, "")
	}
	if ar.RawRegistration != "" {
		return ar.NewError(oidc.ErrorOIDCRegistrationNotSupported, "")
	}

	return nil
}

// Verify checks that the passed parameters match the accociated requirements.
func (ar *AuthenticationRequest) Verify(userID string) error {
	if ar.IDTokenHint != nil {
		// Compare userID with IDTokenHint.
		if userID != ar.IDTokenHint.Claims.(*oidc.IDTokenClaims).Subject {
			return ar.NewError(oidc.ErrorOIDCLoginRequired, "userid mismatch")
		}
	}

	return nil
}

// NewError creates a new error with id and string and the associated request's
// state.
func (ar *AuthenticationRequest) NewError(id string, description string) *AuthenticationError {
	return &AuthenticationError{
		ErrorID:          id,
		ErrorDescription: description,
		State:            ar.State,
	}
}

// NewBadRequest creates a new error with id and string and the associated
// request's state.
func (ar *AuthenticationRequest) NewBadRequest(id string, description string) *AuthenticationBadRequest {
	return &AuthenticationBadRequest{
		ErrorID:          id,
		ErrorDescription: description,
		State:            ar.State,
	}
}

// AuthenticationSuccess holds the outgoind data for a successful OpenID
// Connect 1.0 authorize request as specified at
// http://openid.net/specs/openid-connect-core-1_0.html#AuthResponse and
// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthResponse.
// https://openid.net/specs/openid-connect-session-1_0.html#CreatingUpdatingSessions
type AuthenticationSuccess struct {
	Code        string `url:"code,omitempty"`
	AccessToken string `url:"access_token,omitempty"`
	TokenType   string `url:"token_type,omitempty"`
	IDToken     string `url:"id_token,omitempty"`
	State       string `url:"state"`
	ExpiresIn   int64  `url:"expires_in,omitempty"`

	Scope string `url:"scope,omitempty"`

	SessionState string `url:"session_state,omitempty"`
}

// AuthenticationError holds the outgoind data for a failed OpenID
// Connect 1.0 authorize request as specified at
// http://openid.net/specs/openid-connect-core-1_0.html#AuthError and
// http://openid.net/specs/openid-connect-core-1_0.html#ImplicitAuthError.
type AuthenticationError struct {
	ErrorID          string `url:"error" json:"error"`
	ErrorDescription string `url:"error_description,omitempty" json:"error_description,omitempty"`
	State            string `url:"state,omitempty" json:"state,omitempty"`
}

// Error interface implementation.
func (ae *AuthenticationError) Error() string {
	return ae.ErrorID
}

// Description implements ErrorWithDescription interface.
func (ae *AuthenticationError) Description() string {
	return ae.ErrorDescription
}

// AuthenticationBadRequest holds the outgoing data for a failed OpenID Connect
// 1.0 authorize request with bad request parameters which make it impossible to
// continue with normal auth.
type AuthenticationBadRequest struct {
	ErrorID          string `url:"error" json:"error"`
	ErrorDescription string `url:"error_description,omitempty" json:"error_description,omitempty"`
	State            string `url:"state,omitempty" json:"state,omitempty"`
}

// Error interface implementation.
func (ae *AuthenticationBadRequest) Error() string {
	return ae.ErrorID
}

// Description implements ErrorWithDescription interface.
func (ae *AuthenticationBadRequest) Description() string {
	return ae.ErrorDescription
}
