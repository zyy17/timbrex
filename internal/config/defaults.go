package config

var providerDefaults = map[string]Provider{
	"stub":   {},
	"fish":   {Model: "s2.1-pro", Endpoint: "https://api.fish.audio/v1"},
	"gemini": {Model: "gemini-3.8-flash-tts", Voice: "Kore", Endpoint: "https://generativelanguage.googleapis.com/v1beta"},
	"openai": {
		Model: "gpt-4o-mini-tts", Voice: "coral",
		Endpoint: "https://api.openai.com/v1",
	},
	"elevenlabs": {
		Model: "eleven_v3", Endpoint: "https://api.elevenlabs.io/v1",
	},
	"minimax": {
		Model: "speech-2.8-hd", Voice: "English_expressive_narrator",
		Endpoint: "https://api.minimax.io/v1",
	},
	"azure": {Voice: "en-US-JennyNeural"},
	"qwen":  {Model: "qwen-audio-3.1-tts-flash", Voice: "longanhuan_v3.1"},
}

var envKeys = map[string]string{
	"FISH_API_KEY":          "providers.fish.api_key",
	"FISH_MODEL":            "providers.fish.model",
	"FISH_VOICE_ID":         "providers.fish.voice",
	"FISH_BASE_URL":         "providers.fish.endpoint",
	"GEMINI_API_KEY":        "providers.gemini.api_key",
	"GEMINI_MODEL":          "providers.gemini.model",
	"GEMINI_VOICE":          "providers.gemini.voice",
	"GEMINI_BASE_URL":       "providers.gemini.endpoint",
	"TIMBREX_PROVIDER":      "provider",
	"TIMBREX_STRICT_TAGS":   "strict_tags",
	"OPENAI_API_KEY":        "providers.openai.api_key",
	"OPENAI_MODEL":          "providers.openai.model",
	"OPENAI_VOICE":          "providers.openai.voice",
	"OPENAI_BASE_URL":       "providers.openai.endpoint",
	"ELEVENLABS_API_KEY":    "providers.elevenlabs.api_key",
	"ELEVENLABS_MODEL":      "providers.elevenlabs.model",
	"ELEVENLABS_VOICE_ID":   "providers.elevenlabs.voice",
	"ELEVENLABS_BASE_URL":   "providers.elevenlabs.endpoint",
	"MINIMAX_API_KEY":       "providers.minimax.api_key",
	"MINIMAX_MODEL":         "providers.minimax.model",
	"MINIMAX_VOICE_ID":      "providers.minimax.voice",
	"MINIMAX_BASE_URL":      "providers.minimax.endpoint",
	"AZURE_SPEECH_KEY":      "providers.azure.api_key",
	"AZURE_SPEECH_VOICE":    "providers.azure.voice",
	"AZURE_SPEECH_REGION":   "providers.azure.region",
	"AZURE_SPEECH_ENDPOINT": "providers.azure.endpoint",
	"DASHSCOPE_API_KEY":     "providers.qwen.api_key",
	"QWEN_MODEL":            "providers.qwen.model",
	"QWEN_VOICE":            "providers.qwen.voice",
	"QWEN_ENDPOINT":         "providers.qwen.endpoint",
}

func defaults() map[string]any {
	values := map[string]any{"provider": "stub", "strict_tags": false}
	for id, p := range providerDefaults {
		prefix := "providers." + id + "."
		values[prefix+"api_key"] = p.APIKey
		values[prefix+"model"] = p.Model
		values[prefix+"voice"] = p.Voice
		values[prefix+"endpoint"] = p.Endpoint
		values[prefix+"region"] = p.Region
	}
	return values
}
