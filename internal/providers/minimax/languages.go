package minimax

import (
	"fmt"
	"regexp"
	"strings"
)

// MiniMax accepts language names, not locale codes. Keep language selection
// separate from the optional script/region, which does not select an accent.
// https://platform.minimax.io/docs/api-reference/speech-t2a-http
var languages = map[string]string{
	"af": "Afrikaans", "ar": "Arabic", "bg": "Bulgarian", "ca": "Catalan",
	"cs": "Czech", "da": "Danish", "de": "German", "el": "Greek",
	"en": "English", "es": "Spanish", "fa": "Persian", "fi": "Finnish",
	"fil": "Filipino", "fr": "French", "he": "Hebrew", "hi": "Hindi",
	"hr": "Croatian", "hu": "Hungarian", "id": "Indonesian", "it": "Italian",
	"ja": "Japanese", "ko": "Korean", "ms": "Malay", "nl": "Dutch",
	"nn": "Nynorsk", "no": "Norwegian", "pl": "Polish", "pt": "Portuguese",
	"ro": "Romanian", "ru": "Russian", "sk": "Slovak", "sl": "Slovenian",
	"sv": "Swedish", "ta": "Tamil", "th": "Thai", "tr": "Turkish",
	"uk": "Ukrainian", "vi": "Vietnamese", "yue": "Chinese,Yue", "zh": "Chinese",
}

// Only language[-Script][-REGION] forms are interpreted. Extlangs, variants,
// extensions and private-use suffixes must not silently become another hint.
var languageTag = regexp.MustCompile(`^[a-z]{2,3}(-[a-z]{4})?(-([a-z]{2}|[0-9]{3}))?$`)

func languageBoost(hint, model string) (string, error) {
	if hint == "" {
		return "", nil
	}
	modern := false
	switch model {
	case "speech-2.8-hd", "speech-2.8-turbo", "speech-2.6-hd", "speech-2.6-turbo":
		modern = true
	case "speech-02-hd", "speech-02-turbo", "speech-01-hd", "speech-01-turbo":
	default:
		return "", fmt.Errorf("language hints are not supported for MiniMax model %q", model)
	}
	lang := strings.ToLower(hint)
	if lang == "auto" {
		return "auto", nil
	}
	if !languageTag.MatchString(lang) {
		return "", fmt.Errorf("unsupported language hint %q", hint)
	}
	primary, _, _ := strings.Cut(lang, "-")
	value := languages[primary]
	// Preserve the adapter's existing two exact Cantonese locale aliases.
	if lang == "zh-hk" || lang == "zh-mo" {
		value = "Chinese,Yue"
	}
	if value == "" {
		return "", fmt.Errorf("unsupported language hint %q", hint)
	}
	if !modern && (primary == "fa" || primary == "fil" || primary == "ta") {
		return "", fmt.Errorf("MiniMax language %q requires speech-2.6-hd, speech-2.6-turbo, speech-2.8-hd or speech-2.8-turbo", hint)
	}
	return value, nil
}
