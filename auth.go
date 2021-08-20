package GraphClient

import (
	"encoding/json"
	"errors"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Auth struct {
	Tenant       string
	ClientId     string
	ClientSecret string
	RedirectUrl  string
	Scopes       []string
	ResponseMode string
	TokenCache   ITokenCache
}

type AuthBuilder struct {
	auth *Auth
}

type Token struct {
	AccessToken  string
	Expires      int64
	RefreshToken string
}

func (t *AuthBuilder) WithTenant(Tenant string) *AuthBuilder {
	t.auth.Tenant = Tenant
	return t
}
func (t *AuthBuilder) WithClientId(ClientId string) *AuthBuilder {
	t.auth.ClientId = ClientId
	return t
}
func (t *AuthBuilder) WithClientSecret(ClientSecret string) *AuthBuilder {
	t.auth.ClientSecret = ClientSecret
	return t
}
func (t *AuthBuilder) WithRedirectUrl(RedirectUrl string) *AuthBuilder {
	t.auth.RedirectUrl = RedirectUrl
	return t
}
func (t *AuthBuilder) WithScopes(Scopes []string) *AuthBuilder {
	t.auth.Scopes = Scopes
	return t
}
func (t *AuthBuilder) WithResponseMode(ResponseMode string) *AuthBuilder {
	t.auth.ResponseMode = ResponseMode
	return t
}
func (t *AuthBuilder) WithCustomTokenCache(TokenCache ITokenCache) *AuthBuilder {
	t.auth.TokenCache = TokenCache
	return t
}
func (t *AuthBuilder) Build() (*Auth, error) {
	if t.auth.ClientId == "" ||
		t.auth.ClientSecret == "" ||
		t.auth.Scopes == nil ||
		t.auth.RedirectUrl == "" {
		return nil, errors.New("ClientId,ClientSecret,Scopes,RedirectUrl cannot be empty")
	}
	if t.auth.Tenant == "" {
		t.auth.Tenant = "common"
	}
	if t.auth.ResponseMode == "" {
		t.auth.ResponseMode = "query"
	}
	if t.auth.TokenCache == nil {
		t.auth.TokenCache = &DefaultTokenCache{}
	}
	return t.auth, nil
}
func (t *Auth) GetAuthUrl(state string) string {
	u := url.URL{}
	u.Scheme = "https"
	u.Host = "login.microsoftonline.com"
	u.Path = "/" + t.Tenant + "/oauth2/v2.0/authorize"
	q := u.Query()
	q.Add("client_id", t.ClientId)
	q.Add("response_type", "code")
	q.Add("redirect_uri", t.RedirectUrl)
	q.Add("response_mode", t.ResponseMode)
	var scope string
	for i := 0; i < len(t.Scopes); i++ {
		scope += t.Scopes[i]
		if i != len(t.Scopes)-1 {
			scope += " "
		}
	}
	q.Add("scope", scope)
	if state != "" {
		q.Add("state", state)
	}
	u.RawQuery = q.Encode()
	return u.String()
}

func (t *Auth) GetAccessToken(code string) (string, error) {
	client := &http.Client{}
	reqBody := url.Values{}
	Ret := &Token{}
	reqBody.Add("client_id", t.ClientId)
	reqBody.Add("grant_type", "authorization_code")
	var scope string
	for i := 0; i < len(t.Scopes); i++ {
		scope += t.Scopes[i]
		if i != len(t.Scopes)-1 {
			scope += " "
		}
	}
	reqBody.Add("scope", scope)
	reqBody.Add("code", code)
	reqBody.Add("redirect_uri", t.RedirectUrl)
	reqBody.Add("client_secret", t.ClientSecret)
	req, err := http.NewRequest("POST", "https://login.microsoftonline.com/"+t.Tenant+"/oauth2/v2.0/token", strings.NewReader(reqBody.Encode()))
	if err != nil {
		return "", err
	}
	req.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	at := gjson.Get(string(content), "access_token")
	if !at.Exists() {
		return "", errors.New("Access Token is Empty")
	}
	exp := gjson.Get(string(content), "expires_in")
	if !exp.Exists() {
		return "", errors.New("Expires is Empty")
	}
	rt := gjson.Get(string(content), "refresh_token")
	if !rt.Exists() {
		return "", errors.New("Refresh Token is Empty")
	}
	Ret.AccessToken = at.String()
	Ret.RefreshToken = rt.String()
	Ret.Expires = exp.Int() + time.Now().Unix()
	id, err := t.getHomeAccountId(*Ret)
	if err != nil {
		return "", err
	}
	err = t.TokenCache.Set(id, *Ret)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (t *Auth) getHomeAccountId(token Token) (string, error) {
	request, err := http.NewRequest("GET", "https://graph.microsoft.com/v1.0/me", nil)
	if err != nil {
		return "", err
	}
	request.Header["Authorization"] = []string{"Bearer " + token.AccessToken}
	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	rawBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	m := map[string]interface{}{}
	err = json.Unmarshal(rawBody, &m)
	if err != nil {
		return "", err
	}
	homeAccountId, ok := m["id"].(string)
	if !ok {
		return "", errors.New("HomeAccountId is Invalid")
	}
	return homeAccountId, nil
}

func (t *Auth) RefreshAccessToken(token Token) (Token, error) {
	client := &http.Client{}
	reqBody := url.Values{}
	reqBody.Add("client_id", t.ClientId)
	var scope string
	for i := 0; i < len(t.Scopes); i++ {
		scope += t.Scopes[i]
		if i != len(t.Scopes)-1 {
			scope += " "
		}
	}
	reqBody.Add("scope", scope)
	reqBody.Add("refresh_token", token.RefreshToken)
	reqBody.Add("redirect_uri", t.RedirectUrl)
	reqBody.Add("grant_type", "refresh_token")
	reqBody.Add("client_secret", t.ClientSecret)
	req, err := http.NewRequest("POST", "https://login.microsoftonline.com/common/oauth2/v2.0/token", strings.NewReader(reqBody.Encode()))
	if err != nil {
		return Token{}, err
	}
	req.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	resp, err := client.Do(req)
	if err != nil {
		return Token{}, err
	}
	defer resp.Body.Close()
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return Token{}, err
	}
	at := gjson.Get(string(content), "access_token")
	if !at.Exists() {
		return Token{}, errors.New("Access Token is Empty")
	}
	exp := gjson.Get(string(content), "expires_in")
	if !exp.Exists() {
		return Token{}, errors.New("Expires is Empty")
	}
	rt := gjson.Get(string(content), "refresh_token")
	if !rt.Exists() {
		return Token{}, errors.New("Refresh Token is Empty")
	}
	return Token{
		AccessToken:  at.String(),
		Expires:      exp.Int() + time.Now().Unix(),
		RefreshToken: rt.String(),
	}, nil
}

func (t *Auth) GetValidateToken(token Token) (Token, error, bool) { //bool is used to indicate if Token is refreshed here
	if token.Expires-time.Now().Unix() < 300 {
		ret, err := t.RefreshAccessToken(token)
		return ret, err, true
	} else {
		return token, nil, false
	}
}
