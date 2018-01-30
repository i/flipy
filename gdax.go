package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
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

func (gdx *GdaxClient) placeOrder(productID ProductID, side Side, price Money, size float64) (OrderID, error) {
	body := bytes.NewBuffer(nil)
	if err := json.NewEncoder(body).Encode(orderRequest{
		Size:      fmt.Sprintf("%f", size),
		Price:     price,
		Side:      side,
		ProductID: productID,
		PostOnly:  true,
	}); err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", gdx.baseURL+"/orders", body)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range gdx.auth.Headers("POST", "/orders", body.String()) {
		req.Header.Set(k, v)
	}

	resp, err := gdx.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		return "", fmt.Errorf("bad code: %d\n%s\n", resp.StatusCode, string(b))
	}

	p := new(struct {
		OrderID `json:"id"`
	})
	if err := json.NewDecoder(resp.Body).Decode(p); err != nil {
		return "", err
	}

	return p.OrderID, nil
}

func (gdx *GdaxClient) Buy(productID ProductID, price Money, size float64) (OrderID, error) {
	return gdx.placeOrder(productID, Buy, price, size)
}

func doit() int {
	panic("")
}

func doit2() int {
	log.Fatal("")
	return 0
}

func (gdx *GdaxClient) Sell(productID ProductID, price Money, size float64) (OrderID, error) {
	return gdx.placeOrder(productID, Sell, price, size)
}

func (gdx *GdaxClient) Cancel(orderID OrderID) error {
	path := "/orders/" + string(orderID)
	req, err := http.NewRequest("DELETE", gdx.baseURL+path, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")
	for k, v := range gdx.auth.Headers("DELETE", path, "") {
		req.Header.Set(k, v)
	}

	resp, err := gdx.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		b, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("bad code: %d\n%s\n", resp.StatusCode, string(b))
	}

	return nil

}

type orderRequest struct {
	Size      string    `json:"size"`
	Price     Money     `json:"price"`
	Side      Side      `json:"side"`
	ProductID ProductID `json:"product_id"`
	PostOnly  bool      `json:"post_only"`
	Stp       string    `json:"stp,omitempty"`
}
