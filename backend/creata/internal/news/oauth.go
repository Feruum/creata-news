package news

import (
	"crypto"
	"crypto/hmac"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func (s *Server) oidc(provider string) (Doc, string, string, error) {
	var discover, client, secret string
	switch provider {
	case "google":
		discover = "https://accounts.google.com/.well-known/openid-configuration"
		client = s.Config.GoogleClientID
		secret = s.Config.GoogleClientSecret
	case "telegram":
		discover = "https://oauth.telegram.org/.well-known/openid-configuration"
		client = s.Config.TelegramClientID
		secret = s.Config.TelegramClientSecret
	default:
		return nil, "", "", errStatus(404, "Unknown sign-in provider")
	}
	if client == "" || secret == "" {
		return nil, "", "", errStatus(503, "Sign-in provider is not configured")
	}
	res, e := s.client.Get(discover)
	if e != nil {
		return nil, "", "", errStatus(503, "Sign-in provider unavailable")
	}
	defer res.Body.Close()
	var d Doc
	if res.StatusCode != 200 || json.NewDecoder(res.Body).Decode(&d) != nil {
		return nil, "", "", errStatus(503, "Sign-in provider unavailable")
	}
	return d, client, secret, nil
}
func (s *Server) oauthStart(w http.ResponseWriter, r *http.Request) error {
	provider := r.PathValue("provider")
	cfg, client, _, e := s.oidc(provider)
	if e != nil {
		return e
	}
	state, nonce, verifier := id()+id(), id(), id()+id()
	returnTo, e := s.returnTarget(r.URL.Query().Get("returnTo"))
	if e != nil {
		return e
	}
	if e = s.Store.Write(r.Context(), func(t *Tx) error {
		return t.Put(r.Context(), "oauth", Doc{"id": hash(state), "provider": provider, "nonce": nonce, "verifier": verifier, "returnTo": returnTo, "expiresAt": time.Now().Add(10 * time.Minute).UTC().Format(time.RFC3339)})
	}); e != nil {
		return e
	}
	http.SetCookie(w, &http.Cookie{Name: "creata_oauth_state", Value: state, Domain: s.Config.CookieDomain, Path: base + "/auth/", HttpOnly: true, Secure: strings.HasPrefix(s.Config.PublicURL, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: 600})
	u, e := url.Parse(str(cfg["authorization_endpoint"]))
	if e != nil || u.Scheme != "https" {
		return errStatus(503, "Invalid provider configuration")
	}
	q := u.Query()
	q.Set("client_id", client)
	q.Set("redirect_uri", strings.TrimRight(s.Config.APIURL, "/")+base+"/auth/"+provider+"/callback")
	q.Set("response_type", "code")
	q.Set("scope", "openid profile email")
	q.Set("state", state)
	q.Set("nonce", nonce)
	challenge := sha256.Sum256([]byte(verifier))
	q.Set("code_challenge", base64.RawURLEncoding.EncodeToString(challenge[:]))
	q.Set("code_challenge_method", "S256")
	u.RawQuery = q.Encode()
	http.Redirect(w, r, u.String(), 302)
	return nil
}
func (s *Server) oauthCallback(w http.ResponseWriter, r *http.Request) error {
	state := r.URL.Query().Get("state")
	c, e := r.Cookie("creata_oauth_state")
	if e != nil || state == "" || !hmac.Equal([]byte(state), []byte(c.Value)) {
		return errStatus(400, "Invalid sign-in state")
	}
	flow, e := s.Store.Get(r.Context(), "oauth", hash(state))
	if e != nil || time.Now().After(parseTime(flow["expiresAt"])) || str(flow["provider"]) != r.PathValue("provider") {
		return errStatus(400, "Sign-in session expired")
	}
	// Consume the state before token exchange: callbacks cannot be replayed.
	if e = s.Store.Write(r.Context(), func(t *Tx) error {
		_, e := t.Get(r.Context(), "oauth", hash(state))
		if e != nil {
			return errStatus(400, "Sign-in state already used")
		}
		return t.Delete(r.Context(), "oauth", hash(state))
	}); e != nil {
		return e
	}
	provider := r.PathValue("provider")
	cfg, client, secret, e := s.oidc(provider)
	if e != nil {
		return e
	}
	form := url.Values{"grant_type": {"authorization_code"}, "code": {r.URL.Query().Get("code")}, "client_id": {client}, "client_secret": {secret}, "redirect_uri": {strings.TrimRight(s.Config.APIURL, "/") + base + "/auth/" + provider + "/callback"}, "code_verifier": {str(flow["verifier"])}}
	res, e := s.client.PostForm(str(cfg["token_endpoint"]), form)
	if e != nil {
		return errStatus(503, "Token exchange unavailable")
	}
	defer res.Body.Close()
	var tokens Doc
	if res.StatusCode != 200 || json.NewDecoder(res.Body).Decode(&tokens) != nil {
		return errStatus(401, "Sign-in was not accepted")
	}
	claims, e := s.verifyIDToken(str(tokens["id_token"]), cfg, client, str(flow["nonce"]))
	if e != nil {
		return errStatus(401, "Invalid identity token")
	}
	var u Doc
	e = s.Store.Write(r.Context(), func(t *Tx) error {
		identID := hash(provider + ":" + str(claims["sub"]))
		ident, err := t.Get(r.Context(), "identity", identID)
		if err == nil {
			u, err = t.Get(r.Context(), "user", str(ident["userId"]))
			return err
		}
		uid := id()
		if provider == "telegram" {
			if existing, lookupErr := findTelegram(t, r.Context(), str(claims["sub"])); lookupErr == nil {
				u = existing
				return t.Put(r.Context(), "identity", Doc{"id": identID, "provider": provider, "subject": claims["sub"], "userId": u["id"]})
			}
		}
		u = newUser(uid, str(claims["name"]), "ru")
		if str(u["name"]) == "" {
			u["name"] = "Creata reader"
		}
		if boolean(claims["email_verified"]) {
			u["email"] = str(claims["email"])
		}
		if provider == "telegram" {
			u["telegramId"] = str(claims["sub"])
			u["telegramLinked"] = true
		}
		if err = t.Put(r.Context(), "user", u); err != nil {
			return err
		}
		return t.Put(r.Context(), "identity", Doc{"id": identID, "provider": provider, "subject": claims["sub"], "userId": uid})
	})
	if e != nil {
		return e
	}
	if e = s.session(w, r.Context(), u); e != nil {
		return e
	}
	http.SetCookie(w, &http.Cookie{Name: "creata_oauth_state", Domain: s.Config.CookieDomain, Path: base + "/auth/", MaxAge: -1, HttpOnly: true, Secure: strings.HasPrefix(s.Config.PublicURL, "https://"), SameSite: http.SameSiteLaxMode})
	target := str(flow["returnTo"])
	if strings.HasPrefix(target, "/") {
		target = strings.TrimRight(s.Config.PublicURL, "/") + target
	}
	http.Redirect(w, r, target, 302)
	return nil
}
func (s *Server) returnTarget(raw string) (string, error) {
	if raw == "" {
		return "/", nil
	}
	if strings.ContainsAny(raw, "\\\r\n\x00") {
		return "", errStatus(400, "Invalid return destination")
	}
	u, e := url.Parse(raw)
	if e != nil || u.User != nil {
		return "", errStatus(400, "Invalid return destination")
	}
	if u.Scheme == "" && u.Host == "" && strings.HasPrefix(raw, "/") && !strings.HasPrefix(raw, "//") {
		return raw, nil
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errStatus(400, "Return destination is not allowed")
	}
	origin := u.Scheme + "://" + u.Host
	for _, allowed := range s.Config.Origins {
		if origin == strings.TrimSpace(allowed) {
			return u.String(), nil
		}
	}
	return "", errStatus(400, "Return destination is not allowed")
}
func (s *Server) verifyIDToken(token string, cfg Doc, audience, nonce string) (Doc, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, errors.New("invalid token")
	}
	decodePart := func(v string) (Doc, error) {
		b, e := base64.RawURLEncoding.DecodeString(v)
		var d Doc
		if e == nil {
			e = json.Unmarshal(b, &d)
		}
		return d, e
	}
	header, e := decodePart(parts[0])
	if e != nil || str(header["alg"]) != "RS256" {
		return nil, errors.New("unsupported token signature")
	}
	claims, e := decodePart(parts[1])
	if e != nil || str(claims["iss"]) != str(cfg["issuer"]) || str(claims["sub"]) == "" || str(claims["nonce"]) != nonce || number(claims["exp"]) < float64(time.Now().Unix()) || number(claims["iat"]) > float64(time.Now().Add(time.Minute).Unix()) {
		return nil, errors.New("invalid token claims")
	}
	if str(claims["aud"]) != audience && !has(claims["aud"], audience) {
		return nil, errors.New("invalid audience")
	}
	if len(array(claims["aud"])) > 1 && str(claims["azp"]) != audience {
		return nil, errors.New("invalid authorized party")
	}
	res, e := s.client.Get(str(cfg["jwks_uri"]))
	if e != nil {
		return nil, e
	}
	defer res.Body.Close()
	var keys Doc
	if res.StatusCode != 200 || json.NewDecoder(res.Body).Decode(&keys) != nil {
		return nil, errors.New("keys unavailable")
	}
	sig, e := base64.RawURLEncoding.DecodeString(parts[2])
	if e != nil {
		return nil, e
	}
	sum := sha256.Sum256([]byte(parts[0] + "." + parts[1]))
	for _, raw := range array(keys["keys"]) {
		k := object(raw)
		if str(k["kid"]) != str(header["kid"]) || str(k["kty"]) != "RSA" {
			continue
		}
		n, e1 := base64.RawURLEncoding.DecodeString(str(k["n"]))
		ex, e2 := base64.RawURLEncoding.DecodeString(str(k["e"]))
		if e1 != nil || e2 != nil {
			continue
		}
		exponent := 0
		for _, b := range ex {
			exponent = exponent*256 + int(b)
		}
		if e = rsa.VerifyPKCS1v15(&rsa.PublicKey{N: new(big.Int).SetBytes(n), E: exponent}, crypto.SHA256, sum[:], sig); e == nil {
			return claims, nil
		}
	}
	return nil, errors.New("token signature invalid")
}
