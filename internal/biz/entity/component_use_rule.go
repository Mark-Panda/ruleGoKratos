package entity

import "time"

type ComponentUseRule struct {
	ID            int64      `json:"id"`
	ComponentName string     `json:"componentName"`
	ComponentType string     `json:"componentType"`
	Disabled      bool       `json:"disabled"`
	UseDesc       string     `json:"useDesc"`
	UseRuleDesc   string     `json:"useRuleDesc"`
	CreatedAt     *time.Time `json:"createdAt"`
	UpdatedAt     *time.Time `json:"updatedAt"`
}
