package config

// EdgeAutoGwConfig indicates the edge gateway auto config
type EdgeAutoGwConfig struct {
	// Enable indicates whether enable edge auto gateway
	// default true
	Enable bool `json:"enable,omitempty"`
}

func NewEdgeAutoGwConfig() *EdgeAutoGwConfig {
	return &EdgeAutoGwConfig{
		Enable:    true
	}
}
