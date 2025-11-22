package entity

import "time"

type MdWorkflow struct {
	ID           int64      `json:"id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	Desc         string     `json:"desc"`
	ChainID      string     `json:"chainId"`
	ChainName    string     `json:"chainName"`
	ChainVersion int        `json:"chainVersion"`
	CreatedAt    *time.Time `json:"createdAt"`
	UpdatedAt    *time.Time `json:"updatedAt"`
}
