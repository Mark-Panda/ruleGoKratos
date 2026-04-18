package trace

import "ruleGoKratos/internal/biz/entity"

func cloneTraceEvent(e *entity.TraceEvent) *entity.TraceEvent {
	if e == nil {
		return nil
	}
	c := *e
	if e.Metadata != nil {
		c.Metadata = make(map[string]interface{}, len(e.Metadata))
		for k, v := range e.Metadata {
			c.Metadata[k] = v
		}
	}
	return &c
}
