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

package managers

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/sirupsen/logrus"
	"stash.kopano.io/kgol/rndm"

	"stash.kopano.io/kc/konnect/identifier"
	"stash.kopano.io/kc/konnect/identifier/clients"
	"stash.kopano.io/kc/konnect/identity"
	"stash.kopano.io/kc/konnect/oidc"
	"stash.kopano.io/kc/konnect/oidc/payload"
	"stash.kopano.io/kc/konnect/utils"
)

// KCIdentityManager implements an identity manager which connects to Kopano
// Groupware Core server.
type KCIdentityManager struct {
	signInFormURI string

	identifier *identifier.Identifier
	clients    *clients.Registry
	logger     logrus.FieldLogger
}

// NewKCIdentityManager creates a new KCIdentityManager from the provided
// parameters.
func NewKCIdentityManager(c *identity.Config, i *identifier.Identifier, clients *clients.Registry) *KCIdentityManager {
	im := &KCIdentityManager{
		signInFormURI: c.SignInFormURI.String(),

		identifier: i,
		clients:    clients,
		logger:     c.Logger,
	}

	return im
}

// Authenticate implements the identity.Manager interface.
func (im *KCIdentityManager) Authenticate(ctx context.Context, rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest) (identity.AuthRecord, error) {
	var user *identifier.IdentifiedUser
	var err error

	if identifiedUser, _ := im.identifier.GetUserFromLogonCookie(ctx, req); identifiedUser != nil {
		// TODO(longsleep): Add other user meta data.
		user = identifiedUser
	} else {
		// Not signed in.
		err = ar.NewError(oidc.ErrorOIDCLoginRequired, "KCIdentityManager: not signed in")
	}

	// Check prompt value.
	switch {
	case ar.Prompts[oidc.PromptNone] == true:
		if err != nil {
			// Never show sign-in, directly return error.
			return nil, err
		}
	case ar.Prompts[oidc.PromptLogin] == true:
		if err == nil {
			// Enforce to show sign-in, when signed in.
			err = ar.NewError(oidc.ErrorOIDCLoginRequired, "KCIdentityManager: prompt=login request")
		}
	case ar.Prompts[oidc.PromptSelectAccount] == true:
		if err == nil {
			// Enforce to show sign-in, when signed in.
			err = ar.NewError(oidc.ErrorOIDCLoginRequired, "KCIdentityManager: prompt=select_account request")
		}
	default:
		// Let all other prompt values pass.
	}

	if err != nil {
		u, _ := url.Parse(im.signInFormURI)
		u.RawQuery = fmt.Sprintf("flow=%s&%s", identifier.FlowOIDC, req.URL.RawQuery)
		utils.WriteRedirect(rw, http.StatusFound, u, nil, false)

		return nil, &identity.IsHandledError{}
	}

	auth := NewAuthRecord(user.Subject(), nil, nil)
	auth.SetUser(user)

	return auth, nil
}

// Authorize implements the identity.Manager interface.
func (im *KCIdentityManager) Authorize(ctx context.Context, rw http.ResponseWriter, req *http.Request, ar *payload.AuthenticationRequest, auth identity.AuthRecord) (identity.AuthRecord, error) {
	promptConsent := false
	var approvedScopes map[string]bool

	// Check prompt value.
	switch {
	case ar.Prompts[oidc.PromptConsent] == true:
		promptConsent = true
	default:
		// Let all other prompt values pass.
	}

	clientDetails, err := im.clients.Lookup(req.Context(), ar.ClientID, ar.RedirectURI)
	if err != nil {
		return nil, err
	}

	// If not trusted, always force consent.
	if clientDetails.Trusted {
		approvedScopes = ar.Scopes
	} else {
		promptConsent = true
	}

	// Check given consent.
	consent, err := im.identifier.GetConsentFromConsentCookie(req.Context(), rw, req)
	if err != nil {
		return nil, err
	}
	if consent != nil {
		if !consent.Allow {
			return auth, ar.NewError(oidc.ErrorOAuth2AccessDenied, "consent denied")
		}

		promptConsent = false
		approvedScopes = consent.ApprovedScopes(ar.Scopes)
	}

	if consent == nil {
		// Offline access validation.
		// http://openid.net/specs/openid-connect-core-1_0.html#OfflineAccess
		if ok, _ := approvedScopes[oidc.ScopeOfflineAccess]; ok {
			if !promptConsent {
				// Ensure that the prompt parameter contains consent unless
				// other conditions for processing the request permitting offline
				// access to the requested resources are in place; unless one or
				// both of these conditions are fulfilled, then it MUST ignore the
				// offline_access request,
				delete(approvedScopes, oidc.ScopeOfflineAccess)
				im.logger.Debugln("consent is required for offline access but not given, removed offline_access scope")
			}
		}
	}

	if promptConsent {
		if ar.Prompts[oidc.PromptNone] == true {
			return auth, ar.NewError(oidc.ErrorOIDCInteractionRequired, "consent required")
		}

		u, _ := url.Parse(im.signInFormURI)
		u.RawQuery = fmt.Sprintf("flow=%s&%s", identifier.FlowConsent, req.URL.RawQuery)
		utils.WriteRedirect(rw, http.StatusFound, u, nil, false)

		return nil, &identity.IsHandledError{}
	}

	auth.AuthorizeScopes(approvedScopes)
	return auth, nil
}

// ApproveScopes implements the Backend interface.
func (im *KCIdentityManager) ApproveScopes(ctx context.Context, userid string, audience string, approvedScopes map[string]bool) (string, error) {
	ref := rndm.GenerateRandomString(32)

	// TODO(longsleep): Store generated ref with provided data.
	return ref, nil
}

// ApprovedScopes implements the Backend interface.
func (im *KCIdentityManager) ApprovedScopes(ctx context.Context, userid string, audience string, ref string) (map[string]bool, error) {
	if ref == "" {
		return nil, fmt.Errorf("KCIdentityManager: invalid ref")
	}

	return nil, nil
}

// Fetch implements the identity.Manager interface.
func (im *KCIdentityManager) Fetch(ctx context.Context, sub string, scopes map[string]bool) (identity.AuthRecord, bool, error) {
	user, err := im.identifier.GetUserFromSubject(ctx, sub)
	if err != nil {
		im.logger.WithError(err).Errorln("KCIdentityManager: identifier error")
		return nil, false, fmt.Errorf("KCIdentityManager: identifier error")
	}

	if user == nil {
		return nil, false, fmt.Errorf("KCIdentityManager: no user")
	}

	if user.Subject() != sub {
		return nil, false, fmt.Errorf("KCIdentityManager: wrong user")
	}

	authorizedScopes, claims := authorizeScopes(user, scopes)

	auth := NewAuthRecord(sub, authorizedScopes, claims)
	auth.SetUser(user)

	return auth, true, nil
}

// ScopesSupported implements the identity.Manager interface.
func (im *KCIdentityManager) ScopesSupported() []string {
	return []string{
		oidc.ScopeProfile,
		oidc.ScopeEmail,
	}
}

// ClaimsSupported implements the identity.Manager interface.
func (im *KCIdentityManager) ClaimsSupported() []string {
	return []string{
		oidc.NameClaim,
		oidc.EmailClaim,
	}
}