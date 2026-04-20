package domain

type TextMessage struct {
	Body  string `json:"messageBody"`
	Phone string `json:"originationNumber"`
}
