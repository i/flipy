package main

type GdaxClient struct {
	baseURL string
}

func (a *GdaxClient) makeURL() {
	//	return fmt.Sprintf("%s/
}

func (a *GdaxClient) placeOrder() error {
	//	http.Post(a.makeURL("/orders"))
	return nil
}

type orderRequest struct {
	Size      string `json:"size"`
	Price     string `json:"price"`
	Side      string `json:"side"`
	ProductID string `json:"product_id"`
	Stp       string `json:"stp,omitempty"`
}

type orderResponse struct {
	Id            string `json:"id"`
	Price         string `json:"price"`
	Size          string `json:"size"`
	ProductID     string `json:"product_id"`
	Side          string `json:"side"`
	Stp           string `json:"stp"`
	Type          string `json:"type"`
	TimeInForce   string `json:"time_in_force"`
	PostOnly      bool   `json:"post_only"`
	CreatedAt     string `json:"created_at"`
	FillFees      string `json:"fill_fees"`
	FilledSize    string `json:"filled_size"`
	ExecutedValue string `json:"executed_value"`
	Status        string `json:"status"`
	Settled       bool   `json:"settled"`
}
