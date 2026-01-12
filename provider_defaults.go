package main

type ProviderDefaults struct {
	BaseURL      string
	DefaultModel string
	Models       []string
}

var ProviderDefaultsByName = map[string]ProviderDefaults{
	"minimax": {
		BaseURL:      "https://api.minimax.io/anthropic",
		DefaultModel: "MiniMax-M2.1",
		Models:       []string{"MiniMax-M2.1"},
	},
	"glm": {
		BaseURL:      "https://api.z.ai/api/anthropic",
		DefaultModel: "glm-4.7",
		Models:       []string{"glm-4.7"},
	},
}
