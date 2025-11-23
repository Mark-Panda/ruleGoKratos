package entity

import "time"

type RuleChain struct {
	ID             int64      `json:"id"`
	UserName       string     `json:"userName"`
	Root           bool       `json:"root"`
	Disabled       bool       `json:"disabled"`
	DebugMode      bool       `json:"debugMode"`
	Name           string     `json:"name"`
	RuleChainID    string     `json:"ruleChainId"`
	RuleVersion    int        `json:"ruleVersion"`
	Configuration  string     `json:"configuration"`
	Metadata       string     `json:"metadata"`
	AdditionalInfo string     `json:"additionalInfo"`
	CreatedAt      *time.Time `json:"createdAt"`
	UpdatedAt      *time.Time `json:"updatedAt"`
	DeletedAt      *time.Time `json:"deletedAt"`
}
