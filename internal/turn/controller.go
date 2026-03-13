package turn

import (
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	defaultContinuationWait      = 900 * time.Millisecond
	defaultShortContinuationWait = 650 * time.Millisecond
	defaultRollbackAudioWindow   = 350
	defaultBargeInThreshold      = 0.75
	defaultBargeInFrames         = 3
)

type Action string

const (
	ActionFinalizeUserTurn Action = "finalize_user_turn"
	ActionContinueListen   Action = "continue_listening"
	ActionBackchannel      Action = "backchannel"
	ActionYield            Action = "yield"
)

type Profile string

const (
	ProfileBalanced  Profile = "balanced"
	ProfilePatient   Profile = "patient"
	ProfileAssertive Profile = "assertive"
)

type Signal struct {
	Action             Action `json:"action"`
	Text               string `json:"text,omitempty"`
	Reason             string `json:"reason,omitempty"`
	WaitMS             int    `json:"wait_ms,omitempty"`
	InterruptAssistant bool   `json:"interrupt_assistant,omitempty"`
	RollbackAudioMS    int    `json:"rollback_audio_ms,omitempty"`
}

type Segment struct {
	PriorText            string
	Text                 string
	DurationMS           int
	InterruptedAssistant bool
}

type Metrics struct {
	Profile               Profile        `json:"profile"`
	Actions               map[string]int `json:"actions"`
	SpeechStarts          int            `json:"speech_starts"`
	SpeechOverlapYields   int            `json:"speech_overlap_yields"`
	PlaybackInterruptions int            `json:"playback_interruptions"`
	ContinuationTimeouts  int            `json:"continuation_timeouts"`
	LastAction            string         `json:"last_action,omitempty"`
	LastReason            string         `json:"last_reason,omitempty"`
	PendingText           string         `json:"pending_text,omitempty"`
	PendingTextChars      int            `json:"pending_text_chars"`
	PlayedAudioMS         int            `json:"played_audio_ms"`
	PlaybackActive        bool           `json:"playback_active"`
	BargeInFrames         int            `json:"barge_in_frames"`
	EvalLoggingEnabled    bool           `json:"eval_logging_enabled"`
	UpdatedAtUnixMS       int64          `json:"updated_at_unix_ms"`
	Metadata              map[string]any `json:"metadata,omitempty"`
}

type CallbackPayload struct {
	Signal  *Signal
	Metrics Metrics
}

type Callbacks struct {
	OnAction  func(Signal)
	OnMetrics func(Metrics)
}

type Config struct {
	Profile               Profile
	ContinuationWait      time.Duration
	ShortContinuationWait time.Duration
	RollbackAudioWindowMS int
	BargeInThreshold      float64
	BargeInConsecutive    int
	ShortUnpunctuatedMS   int
	ShortUnpunctuatedMax  int
	FragmentCharsMax      int
	EvalLoggingEnabled    bool
}

type Option func(*Config)

type Controller struct {
	mu             sync.Mutex
	config         Config
	pendingText    string
	playbackActive bool
	playedAudioMS  int
	bargeInFrames  int
	timer          *time.Timer
	callbacks      Callbacks
	closed         bool
	metrics        Metrics
}

var (
	finalPunctuationRE        = regexp.MustCompile(`[.!?][)"'\]]*$`)
	continuationPunctuationRE = regexp.MustCompile(`(?:,|:|;|-|--|\.\.\.)[)"'\]]*$`)
	tokenCleanupRE            = regexp.MustCompile(`[^a-z0-9' -]+`)
)

var hesitationTokens = tokenSet(
	"ah", "eh", "er", "erm", "hmm", "hm", "mm", "mmm", "uh", "uhh", "uhm", "um", "umm",
	"well", "like",
)

var backchannelPhrases = tokenSet(
	"got it",
	"i see",
	"makes sense",
	"mm-hmm",
	"mmhmm",
	"ok",
	"okay",
	"right",
	"sure",
	"thanks",
	"thank you",
	"yeah",
	"yep",
	"yes",
)

var completeShortUtterances = tokenSet(
	"go on",
	"hold on",
	"nevermind",
	"never mind",
	"no",
	"not now",
	"please continue",
	"repeat that",
	"resume",
	"start over",
	"stop",
	"wait",
	"yes",
)

var trailingContinuationTokens = tokenSet(
	"a", "an", "and", "around", "as", "at", "because", "but", "for", "from",
	"if", "in", "into", "like", "my", "of", "on", "or", "so", "that",
	"the", "then", "this", "to", "under", "until", "when", "while", "with", "your",
)

var leadingQuestionTokens = tokenSet(
	"are", "can", "could", "did", "do", "does", "how", "is", "should",
	"what", "when", "where", "who", "why", "will", "would",
)

func WithProfile(profile Profile) Option {
	return func(config *Config) {
		applyProfile(config, profile)
	}
}

func WithEvalLogging(enabled bool) Option {
	return func(config *Config) {
		config.EvalLoggingEnabled = enabled
	}
}

func NewController(callbacks Callbacks, options ...Option) *Controller {
	config := defaultConfig()
	for _, option := range options {
		if option != nil {
			option(&config)
		}
	}
	controller := &Controller{
		config:    config,
		callbacks: callbacks,
		metrics: Metrics{
			Profile:            config.Profile,
			Actions:            map[string]int{},
			EvalLoggingEnabled: config.EvalLoggingEnabled,
			Metadata:           map[string]any{},
		},
	}
	controller.touchMetricsLocked("")
	return controller
}

func (c *Controller) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	c.stopTimerLocked()
}

func (c *Controller) SetProfile(profile Profile) Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	applyProfile(&c.config, profile)
	c.metrics.Profile = c.config.Profile
	c.touchMetricsLocked("profile")
	metrics := c.metricsSnapshotLocked()
	c.emitMetricsLocked(metrics)
	return metrics
}

func (c *Controller) SetEvalLogging(enabled bool) Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.config.EvalLoggingEnabled = enabled
	c.metrics.EvalLoggingEnabled = enabled
	c.touchMetricsLocked("eval_logging")
	metrics := c.metricsSnapshotLocked()
	c.emitMetricsLocked(metrics)
	return metrics
}

func (c *Controller) SnapshotMetrics() Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.metricsSnapshotLocked()
}

func (c *Controller) Reset() Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pendingText = ""
	c.bargeInFrames = 0
	c.stopTimerLocked()
	c.metrics.PendingText = ""
	c.metrics.PendingTextChars = 0
	c.metrics.BargeInFrames = 0
	c.touchMetricsLocked("reset")
	metrics := c.metricsSnapshotLocked()
	c.emitMetricsLocked(metrics)
	return metrics
}

func (c *Controller) UpdatePlayback(playing bool, playedMS int) Metrics {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.playbackActive = playing
	if playedMS >= 0 {
		c.playedAudioMS = playedMS
	}
	if !playing {
		c.bargeInFrames = 0
	}
	c.metrics.PlaybackActive = c.playbackActive
	c.metrics.PlayedAudioMS = c.playedAudioMS
	c.metrics.BargeInFrames = c.bargeInFrames
	c.touchMetricsLocked("playback")
	metrics := c.metricsSnapshotLocked()
	c.emitMetricsLocked(metrics)
	return metrics
}

func (c *Controller) HandleSpeechStart(interruptedAssistant bool) *Signal {
	c.mu.Lock()
	defer c.mu.Unlock()
	if !interruptedAssistant && !c.playbackActive {
		c.bargeInFrames = 0
		c.metrics.BargeInFrames = 0
		c.touchMetricsLocked("speech_start_ignored")
		return nil
	}
	c.metrics.SpeechStarts++
	c.bargeInFrames = 0
	c.metrics.BargeInFrames = 0
	signal := Signal{
		Action:             ActionYield,
		Reason:             "speech_start",
		InterruptAssistant: true,
		RollbackAudioMS:    rollbackAudioMS(c.playedAudioMS, c.config.RollbackAudioWindowMS),
	}
	c.recordActionLocked(signal)
	c.metrics.PlaybackInterruptions++
	c.emitLocked(signal)
	return &signal
}

func (c *Controller) HandleSpeechProbability(prob float64, interruptedAssistant bool) *Signal {
	c.mu.Lock()
	defer c.mu.Unlock()
	if (!interruptedAssistant && !c.playbackActive) || prob < c.config.BargeInThreshold {
		c.bargeInFrames = 0
		c.metrics.BargeInFrames = 0
		c.touchMetricsLocked("speech_prob_reset")
		return nil
	}
	c.bargeInFrames++
	c.metrics.BargeInFrames = c.bargeInFrames
	c.touchMetricsLocked("speech_prob")
	if c.bargeInFrames < c.config.BargeInConsecutive {
		return nil
	}
	c.bargeInFrames = 0
	c.metrics.BargeInFrames = 0
	signal := Signal{
		Action:             ActionYield,
		Reason:             "speech_overlap",
		InterruptAssistant: true,
		RollbackAudioMS:    rollbackAudioMS(c.playedAudioMS, c.config.RollbackAudioWindowMS),
	}
	c.metrics.SpeechOverlapYields++
	c.metrics.PlaybackInterruptions++
	c.recordActionLocked(signal)
	c.emitLocked(signal)
	return &signal
}

func (c *Controller) ConsumeSegment(segment Segment) Signal {
	c.mu.Lock()
	priorText := normalizeText(segment.PriorText)
	if priorText == "" {
		priorText = c.pendingText
	}
	decision := classifySegment(c.config, Segment{
		PriorText:            priorText,
		Text:                 segment.Text,
		DurationMS:           segment.DurationMS,
		InterruptedAssistant: segment.InterruptedAssistant,
	})
	switch decision.Action {
	case ActionContinueListen:
		c.pendingText = decision.Text
		c.metrics.PendingText = c.pendingText
		c.metrics.PendingTextChars = len(c.pendingText)
		c.scheduleFinalizeLocked(decision.WaitMS)
		c.recordActionLocked(decision)
		c.emitLocked(decision)
	case ActionBackchannel:
		c.recordActionLocked(decision)
		c.emitLocked(decision)
	default:
		c.pendingText = ""
		c.metrics.PendingText = ""
		c.metrics.PendingTextChars = 0
		c.stopTimerLocked()
		c.recordActionLocked(decision)
		c.emitLocked(decision)
	}
	c.mu.Unlock()
	return decision
}

func (c *Controller) Flush(reason string) *Signal {
	c.mu.Lock()
	text := normalizeText(c.pendingText)
	if text == "" {
		c.pendingText = ""
		c.metrics.PendingText = ""
		c.metrics.PendingTextChars = 0
		c.stopTimerLocked()
		c.mu.Unlock()
		return nil
	}
	c.pendingText = ""
	c.metrics.PendingText = ""
	c.metrics.PendingTextChars = 0
	c.stopTimerLocked()
	signal := Signal{
		Action: ActionFinalizeUserTurn,
		Text:   text,
		Reason: strings.TrimSpace(reason),
	}
	if signal.Reason == "continuation_timeout" {
		c.metrics.ContinuationTimeouts++
	}
	c.recordActionLocked(signal)
	c.emitLocked(signal)
	c.mu.Unlock()
	return &signal
}

func (c *Controller) scheduleFinalizeLocked(waitMS int) {
	c.stopTimerLocked()
	delay := time.Duration(waitMS) * time.Millisecond
	if delay <= 0 {
		delay = c.config.ContinuationWait
	}
	c.timer = time.AfterFunc(delay, func() {
		c.Flush("continuation_timeout")
	})
}

func (c *Controller) stopTimerLocked() {
	if c.timer == nil {
		return
	}
	c.timer.Stop()
	c.timer = nil
}

func (c *Controller) recordActionLocked(signal Signal) {
	if c.metrics.Actions == nil {
		c.metrics.Actions = map[string]int{}
	}
	c.metrics.Actions[string(signal.Action)]++
	c.metrics.LastAction = string(signal.Action)
	c.metrics.LastReason = strings.TrimSpace(signal.Reason)
	c.metrics.PlaybackActive = c.playbackActive
	c.metrics.PlayedAudioMS = c.playedAudioMS
	c.metrics.BargeInFrames = c.bargeInFrames
	c.metrics.PendingText = c.pendingText
	c.metrics.PendingTextChars = len(c.pendingText)
	c.touchMetricsLocked("action")
}

func (c *Controller) emitLocked(signal Signal) {
	if c.closed {
		return
	}
	actionCallback := c.callbacks.OnAction
	metrics := c.metricsSnapshotLocked()
	metricsCallback := c.callbacks.OnMetrics
	if actionCallback != nil {
		go actionCallback(signal)
	}
	if metricsCallback != nil {
		go metricsCallback(metrics)
	}
}

func (c *Controller) emitMetricsLocked(metrics Metrics) {
	if c.closed || c.callbacks.OnMetrics == nil {
		return
	}
	callback := c.callbacks.OnMetrics
	go callback(metrics)
}

func (c *Controller) metricsSnapshotLocked() Metrics {
	actions := make(map[string]int, len(c.metrics.Actions))
	for key, value := range c.metrics.Actions {
		actions[key] = value
	}
	metadata := make(map[string]any, len(c.metrics.Metadata))
	for key, value := range c.metrics.Metadata {
		metadata[key] = value
	}
	metrics := c.metrics
	metrics.Actions = actions
	metrics.Metadata = metadata
	return metrics
}

func (c *Controller) touchMetricsLocked(reason string) {
	c.metrics.Profile = c.config.Profile
	c.metrics.PlaybackActive = c.playbackActive
	c.metrics.PlayedAudioMS = c.playedAudioMS
	c.metrics.BargeInFrames = c.bargeInFrames
	c.metrics.PendingText = c.pendingText
	c.metrics.PendingTextChars = len(c.pendingText)
	c.metrics.EvalLoggingEnabled = c.config.EvalLoggingEnabled
	if strings.TrimSpace(reason) != "" {
		if c.metrics.Metadata == nil {
			c.metrics.Metadata = map[string]any{}
		}
		c.metrics.Metadata["last_update"] = reason
	}
	c.metrics.UpdatedAtUnixMS = time.Now().UnixMilli()
}

func classifySegment(config Config, segment Segment) Signal {
	priorText := normalizeText(segment.PriorText)
	currentText := normalizeText(segment.Text)
	combinedText := normalizeText(strings.Join(filterNonEmpty(priorText, currentText), " "))
	durationMS := maxInt(0, segment.DurationMS)
	tokens := tokenize(combinedText)

	if combinedText == "" {
		return Signal{
			Action: ActionBackchannel,
			Reason: "empty",
			WaitMS: int(config.ShortContinuationWait / time.Millisecond),
		}
	}

	if priorText == "" && isBackchannel(currentText) && segment.InterruptedAssistant {
		return Signal{
			Action: ActionBackchannel,
			Text:   combinedText,
			Reason: "assistant_backchannel",
			WaitMS: int(config.ShortContinuationWait / time.Millisecond),
		}
	}

	if incompleteReason := looksIncomplete(config, combinedText, currentText, durationMS, tokens); incompleteReason != "" {
		waitMS := config.ContinuationWait
		if len(tokens) <= 2 {
			waitMS = config.ShortContinuationWait
		}
		return Signal{
			Action: ActionContinueListen,
			Text:   combinedText,
			Reason: incompleteReason,
			WaitMS: int(waitMS / time.Millisecond),
		}
	}

	reason := "semantic_completion"
	if finalPunctuationRE.MatchString(combinedText) {
		reason = "terminal_punctuation"
	}
	return Signal{
		Action: ActionFinalizeUserTurn,
		Text:   combinedText,
		Reason: reason,
	}
}

func normalizeText(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func tokenize(text string) []string {
	normalized := strings.ToLower(normalizeText(text))
	if normalized == "" {
		return nil
	}
	cleaned := tokenCleanupRE.ReplaceAllString(normalized, " ")
	tokens := strings.Fields(cleaned)
	if len(tokens) == 0 {
		return nil
	}
	return tokens
}

func lastToken(text string) string {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return ""
	}
	return tokens[len(tokens)-1]
}

func hasUnbalancedClosers(text string) bool {
	counts := map[rune]int{
		'(': 0, ')': 0,
		'[': 0, ']': 0,
		'{': 0, '}': 0,
		'"': 0, '\'': 0,
	}
	runes := []rune(text)
	for idx, ch := range runes {
		if _, ok := counts[ch]; ok {
			if (ch == '\'' || ch == '"') && isEmbeddedWordQuote(runes, idx) {
				continue
			}
			counts[ch]++
		}
	}
	return counts['('] > counts[')'] ||
		counts['['] > counts[']'] ||
		counts['{'] > counts['}'] ||
		counts['"']%2 == 1 ||
		counts['\'']%2 == 1
}

func isEmbeddedWordQuote(runes []rune, idx int) bool {
	if idx <= 0 || idx >= len(runes)-1 {
		return false
	}
	return isWordRune(runes[idx-1]) && isWordRune(runes[idx+1])
}

func isWordRune(ch rune) bool {
	return (ch >= 'a' && ch <= 'z') ||
		(ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9')
}

func isHesitationOnly(text string) bool {
	tokens := tokenize(text)
	if len(tokens) == 0 {
		return false
	}
	for _, token := range tokens {
		if !hesitationTokens[token] {
			return false
		}
	}
	return true
}

func isBackchannel(text string) bool {
	normalized := strings.ToLower(normalizeText(text))
	if normalized == "" {
		return false
	}
	return backchannelPhrases[normalized] || isHesitationOnly(normalized)
}

func looksLikeCompleteQuestion(text string, tokens []string) bool {
	if finalPunctuationRE.MatchString(text) {
		return true
	}
	if len(tokens) < 3 {
		return false
	}
	return leadingQuestionTokens[tokens[0]]
}

func looksLikeCompleteShortUtterance(text string) bool {
	return completeShortUtterances[strings.ToLower(normalizeText(text))]
}

func looksIncomplete(config Config, text, currentText string, durationMS int, tokens []string) string {
	if text == "" {
		return "empty"
	}
	if continuationPunctuationRE.MatchString(text) {
		return "continuation_punctuation"
	}
	if hasUnbalancedClosers(text) {
		return "open_phrase"
	}
	if isHesitationOnly(currentText) {
		return "hesitation"
	}
	tail := lastToken(text)
	if tail != "" && trailingContinuationTokens[tail] {
		return "trailing_connector"
	}
	if len(tokens) <= 2 && !looksLikeCompleteShortUtterance(text) {
		return "too_short"
	}
	if !finalPunctuationRE.MatchString(text) && !looksLikeCompleteQuestion(text, tokens) {
		if durationMS < config.ShortUnpunctuatedMS && len(tokens) <= config.ShortUnpunctuatedMax {
			return "short_unpunctuated"
		}
		if len(currentText) < config.FragmentCharsMax && len(tokens) <= 4 {
			return "fragment"
		}
	}
	return ""
}

func rollbackAudioMS(playedMS int, windowMS int) int {
	if playedMS <= 0 {
		return 0
	}
	if windowMS <= 0 {
		windowMS = defaultRollbackAudioWindow
	}
	if playedMS < windowMS {
		return playedMS
	}
	return windowMS
}

func tokenSet(values ...string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		normalized := strings.ToLower(strings.TrimSpace(value))
		if normalized != "" {
			set[normalized] = true
		}
	}
	return set
}

func filterNonEmpty(values ...string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			out = append(out, value)
		}
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func defaultConfig() Config {
	config := Config{
		Profile:               ProfileBalanced,
		ContinuationWait:      defaultContinuationWait,
		ShortContinuationWait: defaultShortContinuationWait,
		RollbackAudioWindowMS: defaultRollbackAudioWindow,
		BargeInThreshold:      defaultBargeInThreshold,
		BargeInConsecutive:    defaultBargeInFrames,
		ShortUnpunctuatedMS:   900,
		ShortUnpunctuatedMax:  6,
		FragmentCharsMax:      18,
	}
	return config
}

func normalizeProfile(profile Profile) Profile {
	switch strings.ToLower(strings.TrimSpace(string(profile))) {
	case string(ProfilePatient):
		return ProfilePatient
	case string(ProfileAssertive):
		return ProfileAssertive
	default:
		return ProfileBalanced
	}
}

func applyProfile(config *Config, profile Profile) {
	if config == nil {
		return
	}
	switch normalizeProfile(profile) {
	case ProfilePatient:
		config.Profile = ProfilePatient
		config.ContinuationWait = 1200 * time.Millisecond
		config.ShortContinuationWait = 850 * time.Millisecond
		config.RollbackAudioWindowMS = 300
		config.BargeInThreshold = 0.82
		config.BargeInConsecutive = 4
		config.ShortUnpunctuatedMS = 1050
		config.ShortUnpunctuatedMax = 7
		config.FragmentCharsMax = 22
	case ProfileAssertive:
		config.Profile = ProfileAssertive
		config.ContinuationWait = 650 * time.Millisecond
		config.ShortContinuationWait = 450 * time.Millisecond
		config.RollbackAudioWindowMS = 425
		config.BargeInThreshold = 0.68
		config.BargeInConsecutive = 2
		config.ShortUnpunctuatedMS = 700
		config.ShortUnpunctuatedMax = 5
		config.FragmentCharsMax = 14
	default:
		config.Profile = ProfileBalanced
		config.ContinuationWait = defaultContinuationWait
		config.ShortContinuationWait = defaultShortContinuationWait
		config.RollbackAudioWindowMS = defaultRollbackAudioWindow
		config.BargeInThreshold = defaultBargeInThreshold
		config.BargeInConsecutive = defaultBargeInFrames
		config.ShortUnpunctuatedMS = 900
		config.ShortUnpunctuatedMax = 6
		config.FragmentCharsMax = 18
	}
}
