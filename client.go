package GraphClient

import (
	"encoding/json"
	"errors"
	"github.com/tidwall/gjson"
	"io/ioutil"
	"net/http"
	"strings"
	"sync"
)

type Client struct {
	HomeAccountId string
	TokenCache    ITokenCache
	AuthClient    *Auth
	locker        sync.Mutex
}
type GraphResponse struct {
	Body        string
	RawBody     []byte
	Headers     http.Header
	RawResponse *http.Response
}

const baseUrl = "https://graph.microsoft.com/v1.0"

func GetClient(auth *Auth, HomeAccountId string, tokenCache ...ITokenCache) *Client {
	c := &Client{
		HomeAccountId: HomeAccountId,
		AuthClient:    auth,
	}
	if len(tokenCache) == 0 {
		c.TokenCache = &DefaultTokenCache{}
	} else {
		c.TokenCache = tokenCache[0]
	}
	return c
}

func (t *Client) WithHomeAccountId(HomeAccountId string) *Client {
	t.HomeAccountId = HomeAccountId
	return t
}

/*
@title Send Request To Microsoft Graph(V1.0)
@param method string Method to request
@param path string requestUri
@param body string requestBody(Not available when method is Get)
@param header Optical request headers you want to add (Authorization is no need to set manually)
*/
func (t *Client) Request(method string, path string, body string, header ...map[string][]string) (*GraphResponse, error) {
	if t.HomeAccountId == "" {
		return nil, errors.New("HomeAccountId is not specific")
	}
	t.locker.Lock()
	defer t.locker.Unlock()
	token := t.TokenCache.Get(t.HomeAccountId)
	if token == nil {
		return nil, errors.New("cannot get token from token cache")
	}
	validateToken, err, changed := t.AuthClient.GetValidateToken(*token)
	if err != nil {
		return nil, err
	}
	if changed {
		_ = t.TokenCache.Set(t.HomeAccountId, validateToken)
	}
	req, err := http.NewRequest(method, baseUrl+path, strings.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header["Authorization"] = []string{"Bearer " + validateToken.AccessToken}
	for _, s := range header {
		for a, b := range s {
			req.Header[a] = b
		}
	}
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	rec := &GraphResponse{}
	rec.Body = string(respBody)
	rec.RawBody = respBody
	rec.RawResponse = resp
	rec.Headers = resp.Header
	return rec, nil
}
func (t *GraphResponse) ToJson() (map[string]interface{}, error) {
	m := make(map[string]interface{})
	err := json.Unmarshal(t.RawBody, &m)
	if err != nil {
		return nil, err
	}
	return m, nil
}
func (t *GraphResponse) GetJson(path string) gjson.Result {
	return gjson.Get(t.Body, path)
}
