package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type GdaxClient struct {
	baseURL string
	auth    AuthInfo
	client  http.Client
}

type ClientOption func(*GdaxClient)

func WithAuthInfo(a AuthInfo) ClientOption {
	return func(c *GdaxClient) {
		c.auth = a
	}
}

func WithBaseURL(url string) ClientOption {
	return func(c *GdaxClient) {
		c.baseURL = url
	}
}

func NewGdaxClient(opts ...ClientOption) (*GdaxClient, error) {
	client := new(GdaxClient)
	for _, o := range opts {
		o(client)
	}
	return client, nil
}

func (gdx *GdaxClient) buy(productID ProductID, price Money, size float64) (*orderResponse, error) {
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orderRequest{
		Size:      fmt.Sprintf("%f", size),
		Price:     price,
		Side:      "buy",
		ProductID: productID,
	}); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", gdx.baseURL+"/orders", body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range gdx.auth.Headers("POST", "/orders", body.String()) {
		req.Header.Set(k, v)
	}

	resp, err := gdx.client.Do(req)
	if err != nil {
		return nil, err
	}

	bb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	fmt.Println(string(bb))

	r := new(orderResponse)
	if err := json.NewDecoder(resp.Body).Decode(r); err != nil {
		return nil, err
	}

	return r, nil
}

type orderRequest struct {
	Size      string    `json:"size"`
	Price     Money     `json:"price"`
	Side      Side      `json:"side"`
	ProductID ProductID `json:"product_id"`
	Stp       string    `json:"stp,omitempty"`
}

type orderResponse struct {
	ID            OrderID `json:"id"`
	Price         string  `json:"price"`
	Size          string  `json:"size"`
	ProductID     string  `json:"product_id"`
	Side          string  `json:"side"`
	Stp           string  `json:"stp"`
	Type          string  `json:"type"`
	TimeInForce   string  `json:"time_in_force"`
	PostOnly      bool    `json:"post_only"`
	CreatedAt     string  `json:"created_at"`
	FillFees      string  `json:"fill_fees"`
	FilledSize    string  `json:"filled_size"`
	ExecutedValue string  `json:"executed_value"`
	Status        string  `json:"status"`
	Settled       bool    `json:"settled"`
}
