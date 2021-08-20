package GraphClient

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
)

type ITokenCache interface {
	Get(HomeAccountId string) *Token
	Set(HomeAccountId string, token Token) error
}

type DefaultTokenCache struct {
	locker sync.Mutex
}

func (t *DefaultTokenCache) Get(HomeAccountId string) *Token {
	t.locker.Lock()
	defer t.locker.Unlock()
	file, err := ioutil.ReadFile("token.cache")
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
	file, err := ioutil.ReadFile("token.cache")
	if err != nil {
		return err
	}
	m := map[string]string{}
	if err := json.Unmarshal(file, &m); err != nil {
		return err
	}
	jresult, err := json.Marshal(token)
	if err != nil {
		return err
	}
	m[HomeAccountId] = string(jresult)
	result, err := json.Marshal(m)
	if err != nil {
		return err
	}
	if isExist("token.cache") {
		f, err := os.OpenFile("token.cache", os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0755)
		defer f.Close()
		if err != nil {
			return err
		}
		writer := bufio.NewWriter(f)
		_, err = writer.Write(result)
		if err != nil {
			return err
		}
		err = writer.Flush()
		return err
	} else {
		err := ioutil.WriteFile("token.cache", result, 0755)
		return err
	}
}

func isExist(path string) bool {
	_, err := os.Stat(path)
	if err != nil {
		if os.IsExist(err) {
			return true
		}
		if os.IsNotExist(err) {
			return false
		}
		fmt.Println(err)
		return false
	}
	return true
}
