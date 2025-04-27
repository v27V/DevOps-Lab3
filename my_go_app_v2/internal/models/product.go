package models

// Product представляет строительный товар в каталоге
type Product struct {
	ID          int     `json:"id"`
	Name        string  `json:"name"`
	Category    string  `json:"category"`
	Price       float64 `json:"price"`
	Description string  `json:"description"`
	InStock     bool    `json:"in_stock"`
	Supplier    string  `json:"supplier"`
}
