package tts

import (
	"fmt"
	"log/slog"
	"regexp"
	"runtime"
	"strings"
	"sync/atomic"
	"time"

	ole "github.com/go-ole/go-ole"
	"github.com/go-ole/go-ole/oleutil"
	"golang.org/x/sys/windows"
)

// reCustomEmoji matches Bilibili-style custom emojis like [爱心] or [xxx].
var reCustomEmoji = regexp.MustCompile(`\[[^\]]+\]`)

// reUnicodeEmoji matches standard Unicode emoji codepoints.
var reUnicodeEmoji = regexp.MustCompile(
	`[\x{1F600}-\x{1F64F}` + // emoticons
		`\x{1F300}-\x{1F5FF}` + // misc symbols & pictographs
		`\x{1F680}-\x{1F6FF}` + // transport & map
		`\x{1F700}-\x{1F7FF}` + // alchemical
		`\x{1F800}-\x{1F8FF}` + // supplemental arrows-C
		`\x{1F900}-\x{1F9FF}` + // supplemental symbols
		`\x{1FA00}-\x{1FAFF}` + // symbols extended-A
		`\x{2600}-\x{26FF}` + // misc symbols
		`\x{2700}-\x{27BF}` + // dingbats
		`\x{FE00}-\x{FE0F}` + // variation selectors
		`\x{1F1E0}-\x{1F1FF}` + // regional indicator (flags)
		`\x{200D}` + // zero-width joiner
		`\x{20E3}` + // combining enclosing keycap
		`\x{FE20}-\x{FE2F}` + // combining half marks
		`]`,
)

// sanitize removes emojis and custom emoji tokens from text before TTS.
func sanitize(text string) string {
	text = reCustomEmoji.ReplaceAllString(text, "")
	text = reUnicodeEmoji.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}

// Priority determines speech ordering in the queue.
type Priority int

const (
	PriorityNormal Priority = iota
	PriorityHigh
)

// svsfDefault is the SAPI5 SVSFDefault flag: synchronous, blocking speech.
const svsfDefault = 0

// queueEntry pairs a speech text with the unix timestamp of the originating chat event.
type queueEntry struct {
	text      string
	timestamp int64 // unix seconds from the SSE event
}

// TTSQueue is a two-priority speech queue backed by SAPI5 SpVoice.
type TTSQueue struct {
	highCh   chan queueEntry
	normalCh chan queueEntry
	cfgPtr   atomic.Pointer[Config] // hot-reloadable; always non-nil
	reloadCh chan struct{}           // signals the SAPI5 goroutine to re-apply voice settings
}

// NewTTSQueue creates a queue with the given config.
func NewTTSQueue(cfg *Config) *TTSQueue {
	q := &TTSQueue{
		highCh:   make(chan queueEntry, 20),
		normalCh: make(chan queueEntry, cfg.MaxQueue),
		reloadCh: make(chan struct{}, 1),
	}
	q.cfgPtr.Store(cfg)
	return q
}

// UpdateConfig atomically replaces the running config and signals the SAPI5
// goroutine to re-apply voice settings (rate, volume, voice selection).
// Note: MaxQueue (channel buffer size) takes effect only on restart.
func (q *TTSQueue) UpdateConfig(cfg Config) {
	q.cfgPtr.Store(&cfg)
	select {
	case q.reloadCh <- struct{}{}:
	default: // signal already pending
	}
}

// Enqueue adds text to the queue at the given priority.
// timestamp is the unix-second timestamp of the originating chat event.
// When either queue is full, the oldest item is dropped to make room.
func (q *TTSQueue) Enqueue(text string, priority Priority, timestamp int64) {
	text = sanitize(text)
	if text == "" {
		return
	}
	entry := queueEntry{text: text, timestamp: timestamp}

	switch priority {
	case PriorityHigh:
		select {
		case q.highCh <- entry:
			slog.Debug("tts: enqueued high-priority", "text", truncate(text, 60))
		default:
			// Drop oldest high-priority item to make room.
			select {
			case dropped := <-q.highCh:
				slog.Warn("tts: dropped oldest high-priority (queue full)", "text", truncate(dropped.text, 40))
			default:
			}
			select {
			case q.highCh <- entry:
			default:
			}
		}
	default:
		select {
		case q.normalCh <- entry:
			slog.Debug("tts: enqueued normal", "text", truncate(text, 60))
		default:
			// Drop oldest normal item to make room for the new one.
			select {
			case dropped := <-q.normalCh:
				slog.Warn("tts: dropped oldest normal (queue full)", "text", truncate(dropped.text, 40), "cap", cap(q.normalCh))
			default:
			}
			select {
			case q.normalCh <- entry:
			default:
			}
		}
	}
}

// Run initializes COM/SAPI5 and processes the speech queue.
// It must be called in its own goroutine and blocks until ctx is cancelled.
// It idles (sleeping) until TTS is enabled, so hot-reload works correctly.
func (q *TTSQueue) Run(ctx interface{ Done() <-chan struct{} }) {
	slog.Info("tts: queue Run started, waiting for TTS to be enabled")
	// Wait until TTS is enabled before initialising SAPI5.
	for !q.cfgPtr.Load().Enabled {
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
	slog.Info("tts: TTS enabled, initialising SAPI5")
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := ole.CoInitializeEx(0, ole.COINIT_APARTMENTTHREADED); err != nil {
		// S_FALSE (1) means COM was already initialized on this thread — that's OK.
		if oleErr, ok := err.(*ole.OleError); !ok || oleErr.Code() != 1 {
			slog.Error("tts: CoInitializeEx", "err", err)
			return
		}
	}
	defer ole.CoUninitialize()

	unknown, err := oleutil.CreateObject("SAPI.SpVoice")
	if err != nil {
		slog.Error("tts: CreateObject SAPI.SpVoice", "err", err)
		return
	}
	defer unknown.Release()

	voice, err := unknown.QueryInterface(ole.IID_IDispatch)
	if err != nil {
		slog.Error("tts: QueryInterface IDispatch", "err", err)
		return
	}
	defer voice.Release()

	logAvailableVoices(voice)

	if err := q.applyVoiceSettings(voice); err != nil {
		slog.Warn("tts: voice settings", "err", err)
	}

	slog.Info("tts: SAPI5 ready")
	q.loop(ctx, voice)
}

// logAvailableVoices prints all installed SAPI5 voices in a human-readable format.
func logAvailableVoices(voice *ole.IDispatch) {
	tokensVar, err := oleutil.CallMethod(voice, "GetVoices", "", "")
	if err != nil {
		slog.Warn("tts: GetVoices for listing", "err", err)
		return
	}
	tokens := tokensVar.ToIDispatch()
	if tokens == nil {
		return
	}
	defer tokens.Release()

	countVar, err := oleutil.GetProperty(tokens, "Count")
	if err != nil {
		return
	}
	count := int(countVar.Val)

	slog.Debug("tts: available voices", "count", count)
	for i := range count {
		itemVar, err := oleutil.CallMethod(tokens, "Item", i)
		if err != nil {
			continue
		}
		token := itemVar.ToIDispatch()
		if token == nil {
			continue
		}

		id := ""
		name := ""
		lang := ""

		if v, err := oleutil.GetProperty(token, "Id"); err == nil {
			id = v.ToString()
		}
		if v, err := oleutil.GetProperty(token, "GetDescription"); err == nil {
			name = v.ToString()
		} else if v, err := oleutil.CallMethod(token, "GetDescription"); err == nil {
			name = v.ToString()
		}
		// Language is stored under Attributes sub-object.
		if attrsVar, err := oleutil.GetProperty(token, "Attributes"); err == nil {
			if attrs := attrsVar.ToIDispatch(); attrs != nil {
				if v, err := oleutil.CallMethod(attrs, "GetAttribute", "Language"); err == nil {
					lang = v.ToString()
				}
				attrs.Release()
			}
		}

		slog.Debug("tts: voice", "id", id, "name", name, "lang", lang)
		token.Release()
	}
}

// applyVoiceSettings sets rate, volume, and optionally selects a voice by ID.
func (q *TTSQueue) applyVoiceSettings(voice *ole.IDispatch) error {
	cfg := q.cfgPtr.Load()
	if _, err := oleutil.PutProperty(voice, "Rate", cfg.Rate); err != nil {
		return fmt.Errorf("set Rate: %w", err)
	}
	if _, err := oleutil.PutProperty(voice, "Volume", cfg.Volume); err != nil {
		return fmt.Errorf("set Volume: %w", err)
	}
	voiceID := cfg.VoiceID
	if voiceID == "" {
		voiceID = "ZH-CN" // default to Chinese
	}
	if err := q.selectVoiceByID(voice, voiceID); err != nil {
		slog.Warn("tts: could not set voice, using default", "voiceId", voiceID, "err", err)
	}
	return nil
}

// selectVoiceByID finds the first voice token whose ID contains idSubstr and activates it.
// If idSubstr is "ZH-CN" it also tries SAPI Language attribute filtering as a fallback.
func (q *TTSQueue) selectVoiceByID(voice *ole.IDispatch, idSubstr string) error {
	// First try SAPI attribute-based filtering (e.g. Language=804 for zh-CN).
	if strings.EqualFold(idSubstr, "ZH-CN") {
		if err := q.selectVoiceByAttr(voice, "Language=804"); err == nil {
			return nil
		}
	}
	tokensVar, err := oleutil.CallMethod(voice, "GetVoices", "", "")
	if err != nil {
		return fmt.Errorf("GetVoices: %w", err)
	}
	tokens := tokensVar.ToIDispatch()
	if tokens == nil {
		return fmt.Errorf("GetVoices returned nil")
	}
	defer tokens.Release()

	countVar, err := oleutil.GetProperty(tokens, "Count")
	if err != nil {
		return fmt.Errorf("count: %w", err)
	}
	count := int(countVar.Val)

	for i := range count {
		itemVar, err := oleutil.CallMethod(tokens, "Item", i)
		if err != nil {
			continue
		}
		token := itemVar.ToIDispatch()
		if token == nil {
			continue
		}

		idVar, err := oleutil.GetProperty(token, "Id")
		if err != nil {
			token.Release()
			continue
		}
		id := idVar.ToString()

		if strings.Contains(strings.ToLower(id), strings.ToLower(idSubstr)) {
			_, setErr := oleutil.PutPropertyRef(voice, "Voice", token)
			token.Release()
			if setErr != nil {
				return fmt.Errorf("set Voice: %w", setErr)
			}
			slog.Info("tts: using voice", "id", id)
			return nil
		}
		token.Release()
	}
	return fmt.Errorf("no voice with ID containing %q", idSubstr)
}

// selectVoiceByAttr selects the first voice matching a SAPI required-attributes string
// (e.g. "Language=804" for Simplified Chinese).
func (q *TTSQueue) selectVoiceByAttr(voice *ole.IDispatch, requiredAttrs string) error {
	tokensVar, err := oleutil.CallMethod(voice, "GetVoices", requiredAttrs, "")
	if err != nil {
		return fmt.Errorf("GetVoices(%q): %w", requiredAttrs, err)
	}
	tokens := tokensVar.ToIDispatch()
	if tokens == nil {
		return fmt.Errorf("GetVoices(%q) returned nil", requiredAttrs)
	}
	defer tokens.Release()

	countVar, err := oleutil.GetProperty(tokens, "Count")
	if err != nil {
		return fmt.Errorf("count: %w", err)
	}
	if int(countVar.Val) == 0 {
		return fmt.Errorf("no voices matching %q", requiredAttrs)
	}

	itemVar, err := oleutil.CallMethod(tokens, "Item", 0)
	if err != nil {
		return fmt.Errorf("Item(0): %w", err)
	}
	token := itemVar.ToIDispatch()
	if token == nil {
		return fmt.Errorf("Item(0) returned nil")
	}
	defer token.Release()

	idVar, _ := oleutil.GetProperty(token, "Id")
	if _, err := oleutil.PutPropertyRef(voice, "Voice", token); err != nil {
		return fmt.Errorf("set Voice: %w", err)
	}
	slog.Info("tts: using voice", "id", idVar.ToString(), "attrFilter", requiredAttrs)
	return nil
}

// loop is the main speech dispatch loop, preferring high-priority items.
// If max_age_seconds > 0, entries older than that threshold are skipped silently.
// It also listens on reloadCh to re-apply voice settings after a config update.
func (q *TTSQueue) loop(ctx interface{ Done() <-chan struct{} }, voice *ole.IDispatch) {
	for {
		// Check for a pending config reload first (non-blocking).
		select {
		case <-q.reloadCh:
			if err := q.applyVoiceSettings(voice); err != nil {
				slog.Warn("tts: reload voice settings", "err", err)
			}
		default:
		}

		var entry queueEntry
		// Prefer high-priority without blocking; fall through if empty.
		select {
		case entry = <-q.highCh:
		case <-ctx.Done():
			return
		default:
			select {
			case entry = <-q.highCh:
			case entry = <-q.normalCh:
			case <-q.reloadCh:
				if err := q.applyVoiceSettings(voice); err != nil {
					slog.Warn("tts: reload voice settings", "err", err)
				}
				continue
			case <-ctx.Done():
				return
			}
		}

		// Skip stale entries when max_age_seconds is configured.
		if maxAge := q.cfgPtr.Load().MaxAgeSeconds; maxAge > 0 && entry.timestamp > 0 {
			age := time.Now().Unix() - entry.timestamp
			if age > int64(maxAge) {
				slog.Debug("tts: skipping stale message", "age_s", age, "max_s", maxAge, "text", truncate(entry.text, 40))
				continue
			}
		}

		slog.Info("tts: speaking", "text", truncate(entry.text, 80))
		if _, err := oleutil.CallMethod(voice, "Speak", entry.text, svsfDefault); err != nil {
			// Ignore "operation was cancelled" errors that can occur on shutdown.
			code := windows.Errno(0)
			if !strings.Contains(err.Error(), "0x80045002") {
				slog.Error("tts: Speak", "err", err, "code", code)
			}
		}
	}
}
