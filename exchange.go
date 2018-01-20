package main

type Exchange interface {
	Buy() (OrderID, error)
	Sell() (OrderID, error)
}
