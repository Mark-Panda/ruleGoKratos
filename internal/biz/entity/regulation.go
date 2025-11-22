package entity

import "time"

type Regulation struct {
	ID          int64      `json:"id"`
	UserName    string     `json:"userName"`
	Root        bool       `json:"root"`
	Disabled    bool       `json:"disabled"`
	Name        string     `json:"name"`
	RuleChainID string     `json:"ruleChainId"`
	RuleVersion int        `json:"ruleVersion"`
	RuleConfig  string     `json:"ruleConfig"`
	CreatedAt   *time.Time `json:"createdAt"`
	UpdatedAt   *time.Time `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt"`
}
