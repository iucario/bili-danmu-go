package tts

import (
	"strings"

	"github.com/iucario/bili-danmu-go/internal/config"
)

// Config is an alias for the TTS section of the main application config.
type Config = config.TTSConfig

// applyTemplate replaces {key} placeholders in tmpl with values from vars.
func applyTemplate(tmpl string, vars map[string]string) string {
	for k, v := range vars {
		tmpl = strings.ReplaceAll(tmpl, "{"+k+"}", v)
	}
	return tmpl
}