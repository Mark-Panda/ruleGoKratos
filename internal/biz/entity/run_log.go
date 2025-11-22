package entity

import "time"

type RunLog struct {
	ID             int64      `json:"id"`
	RunID          string     `json:"runId"`
	ChainID        string     `json:"chainId"`
	ChainName      string     `json:"chainName"`
	NodeLog        string     `json:"nodeLog"`
	AdditionalInfo string     `json:"additionalInfo"`
	RuleChainInfo  string     `json:"ruleChainInfo"`
	Metadata       string     `json:"metadata"`
	StartTs        int64      `json:"startTs"`
	EndTs          int64      `json:"endTs"`
	CreatedAt      *time.Time `json:"createdAt"`
	UpdatedAt      *time.Time `json:"updatedAt"`
}
