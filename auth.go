package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"strconv"
	"time"
)

type AuthInfo struct {
	Key        string
	Passphrase string
	Secret     string
}

func (a AuthInfo) signature(method, path, body string) (time.Time, string) {
	ts := time.Now()
	what := []byte(strconv.FormatInt(ts.Unix(), 10) + method + path + body)
	secret, err := base64.StdEncoding.DecodeString(a.Secret)
	if err != nil {
		panic(err)
	}
	h := hmac.New(sha256.New, secret)
	h.Write(what)
	return ts, base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (a AuthInfo) Headers(method, path, body string) map[string]string {
	ts, sig := a.signature(method, path, body)
	return map[string]string{
		"CB-ACCESS-KEY":        a.Key,
		"CB-ACCESS-SIGN":       sig,
		"CB-ACCESS-TIMESTAMP":  strconv.FormatInt(ts.Unix(), 10),
		"CB-ACCESS-PASSPHRASE": a.Passphrase,
	}
}
