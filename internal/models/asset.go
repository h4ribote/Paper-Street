package models

type Asset struct {
	ID     int64  `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Sector string `json:"sector"`
}
