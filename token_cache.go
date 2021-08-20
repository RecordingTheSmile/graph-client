package GraphClient

import (
	"encoding/json"
	"io/ioutil"
	"sync"
)

type ITokenCache interface {
	Get(HomeAccountId string) *Token
	Set(HomeAccountId string, token Token) error
	Delete(HomeAccountId string) error
}

type DefaultTokenCache struct {
	locker sync.Mutex
}

func (t *DefaultTokenCache) Get(HomeAccountId string) *Token {
	t.locker.Lock()
	defer t.locker.Unlock()
	file, err := ioutil.ReadFile("./token.cache")
	if err != nil {
		return nil
	}
	m := map[string]string{}
	if err := json.Unmarshal(file, &m); err != nil {
		return nil
	}
	if m[HomeAccountId] == "" {
		return nil
	}
	result := &Token{}
	if err := json.Unmarshal([]byte(m[HomeAccountId]), result); err != nil {
		return nil
	}
	return result
}

func (t *DefaultTokenCache) Set(HomeAccountId string, token Token) error {
	t.locker.Lock()
	defer t.locker.Unlock()
	file, _ := ioutil.ReadFile("./token.cache")
	m := map[string]string{}
	_ = json.Unmarshal(file, &m)
	jresult, err := json.Marshal(token)
	if err != nil {
		return err
	}
	m[HomeAccountId] = string(jresult)
	result, err := json.Marshal(m)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("./token.cache", result, 0755)
	return err
}

func (t *DefaultTokenCache) Delete(HomeAccountId string) error {
	t.locker.Lock()
	defer t.locker.Unlock()
	file, _ := ioutil.ReadFile("./token.cache")
	m := map[string]string{}
	_ = json.Unmarshal(file, &m)
	delete(m, HomeAccountId)
	result, err := json.Marshal(m)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("./token.cache", result, 0755)
	return err
}
