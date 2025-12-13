package entity

type PlannerTpl struct {
	// NodeUseList []NodeUseRuleTpl `json:"nodeUseList"`
}

type NodeUseRuleTpl struct {
	Type string `json:"type"`
	Desc string `json:"desc"`
}

type NodeToolTpl struct {
	Type           string                 `json:"type"`
	Desc           string                 `json:"desc"`
	Config         map[string]interface{} `json:"config"`
	AdditionalInfo map[string]interface{} `json:"additionalInfo"`
}

type ConnectUseRuleTpl struct {
	FromId string `json:"fromId"`
	ToId   string `json:"toId"`
	Type   string `json:"type"`
}

type AssemblyTpl struct {
}

type ExecuteTpl struct {
}
