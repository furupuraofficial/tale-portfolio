package realtime

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gordonklaus/portaudio"
	"github.com/gorilla/websocket"
	openai "github.com/sashabaranov/go-openai"
)

const (
	RealtimeAPIURL   = "wss://api.openai.com/v1/realtime?model=gpt-realtime"
	InputSampleRate  = 24000
	OutputSampleRate = 24000
	InputChannels    = 1
	OutputChannels   = 1

	// 24kHz × 200ms = 4,800 samples
	RealtimeFrameSamples = 4800
	BytesPerSample       = 2
	RealtimeFrameBytes   = RealtimeFrameSamples * BytesPerSample

	// Conversation history file
	HistoryFile = "data/conversation_history.json"
)

type Client struct {
	conn        *websocket.Conn
	apiKey      string
	inputStream *portaudio.Stream

	inputBuffer []int16
	audioQueue  chan []byte

	transcriptChan chan string
	responseChan   chan string

	stopChan chan struct{}

	// State management
	mu         sync.Mutex
	isPlaying  bool // Whether the assistant is currently playing audio
	micEnabled bool // Whether mic input is allowed to send
	// Whether we intentionally suppress Realtime responses (e.g., during script/TTS)
	suppressRealtime bool
	// Preferred language from frontend (empty = auto-detect)
	preferredLanguage string
	// Track the latest active response id from Realtime API so we can cancel it if needed
	activeResponseID string
	// Whether the Realtime API is allowed to auto-generate responses (server VAD create_response)
	allowRealtimeResponse bool

	// Debug: dump all incoming Realtime events
	debugRealtime bool

	conversationHistory []ConversationItem
	historySubs         []chan ConversationItem

	// Conversation state machine
	currentStep Step
	mode        Mode
	userName    string

	// Channel for AR responses to frontend
	arResponseChan chan ChatAndARResponse

	// Audio stream subscribers (SSE)
	audioSubs []chan ChatAndARResponse

	// OpenAI client for TTS (Audio API)
	ttsClient *openai.Client

	// Transcript streaming
	transcriptSubs    []chan TranscriptEvent
	pendingTranscript string

	// Transcript buffer for batched sending
	transcriptBuffer     strings.Builder
	transcriptBufferLock sync.Mutex
	transcriptFlushTimer *time.Timer
}

type ConversationItem struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// RealtimeEvent represents events sent to/from the Realtime API
type RealtimeEvent struct {
	Type         string                 `json:"type"`
	EventID      string                 `json:"event_id,omitempty"`
	Session      *Session               `json:"session,omitempty"`
	Item         *Item                  `json:"item,omitempty"`
	Audio        string                 `json:"audio,omitempty"`
	Delta        string                 `json:"delta,omitempty"`
	Transcript   string                 `json:"transcript,omitempty"`
	Text         string                 `json:"text,omitempty"`
	ResponseID   string                 `json:"response_id,omitempty"`
	ItemID       string                 `json:"item_id,omitempty"`
	ContentIndex int                    `json:"content_index,omitempty"`
	Response     *Response              `json:"response,omitempty"`
	Error        map[string]interface{} `json:"error,omitempty"`
}

type Session struct {
	Modalities              []string    `json:"modalities,omitempty"`
	Instructions            string      `json:"instructions,omitempty"`
	Voice                   string      `json:"voice,omitempty"`
	InputAudioFormat        string      `json:"input_audio_format,omitempty"`
	OutputAudioFormat       string      `json:"output_audio_format,omitempty"`
	InputAudioTranscription interface{} `json:"input_audio_transcription,omitempty"`
	TurnDetection           interface{} `json:"turn_detection,omitempty"`
}

func defaultTurnDetection(createResponse bool) map[string]interface{} {
	return map[string]interface{}{
		"type":                "server_vad",
		"threshold":           0.5,
		"prefix_padding_ms":   300,
		"silence_duration_ms": 500,
		"create_response":     createResponse,
		"interrupt_response":  true,
	}
}

type Item struct {
	ID      string        `json:"id,omitempty"`
	Type    string        `json:"type,omitempty"`
	Role    string        `json:"role,omitempty"`
	Content []ItemContent `json:"content,omitempty"`
}

type ItemContent struct {
	Type       string `json:"type,omitempty"`
	Text       string `json:"text,omitempty"`
	Audio      string `json:"audio,omitempty"`
	Transcript string `json:"transcript,omitempty"`
}

type Response struct {
	ID           string `json:"id,omitempty"`
	Status       string `json:"status,omitempty"`
	Instructions string `json:"instructions,omitempty"`
}

// TranscriptEvent is sent to frontend subscribers via SSE.
type TranscriptEvent struct {
	Text string `json:"text"`
	Step Step   `json:"step"`
	Done bool   `json:"done"`
}

func NewClient(apiKey string) (*Client, error) {
	if err := portaudio.Initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize portaudio: %w", err)
	}

	client := &Client{
		apiKey:                apiKey,
		inputBuffer:           make([]int16, RealtimeFrameSamples),
		audioQueue:            make(chan []byte, 100),
		transcriptChan:        make(chan string, 10),
		responseChan:          make(chan string, 10),
		stopChan:              make(chan struct{}),
		conversationHistory:   make([]ConversationItem, 0),
		currentStep:           StepIntro,
		mode:                  ModeScript,
		suppressRealtime:      true, // Script steps should be TTS-only by default
		allowRealtimeResponse: false,
		preferredLanguage:     "",
		arResponseChan:        make(chan ChatAndARResponse, 10),
		historySubs:           make([]chan ConversationItem, 0),
		audioSubs:             make([]chan ChatAndARResponse, 0),
		transcriptSubs:        make([]chan TranscriptEvent, 0),
		pendingTranscript:     "",

		// TTS client
		ttsClient: openai.NewClient(apiKey),
	}

	if os.Getenv("REALTIME_DEBUG") == "1" {
		client.debugRealtime = true
		log.Println("[Realtime debug] Enabled raw event logging")
	}

	return client, nil
}

// ==== Playing flag getter / setter ====

func (rc *Client) setPlaying(v bool) {
	rc.mu.Lock()
	changed := rc.isPlaying != v
	rc.isPlaying = v
	rc.mu.Unlock()
	if changed {
		log.Printf("[STATE] playing=%v", v)
	}
}

func (rc *Client) getPlaying() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.isPlaying
}

// Mic flag setter/getter with logging
func (rc *Client) SetMicEnabled(enabled bool) {
	rc.mu.Lock()
	changed := rc.micEnabled != enabled
	rc.micEnabled = enabled
	rc.mu.Unlock()
	if changed {
		log.Printf("[STATE] micEnabled=%v", enabled)
	}
}

func (rc *Client) IsMicEnabled() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.micEnabled
}

// suppressRealtime guards whether we allow the Realtime API to generate responses.
func (rc *Client) setSuppressRealtime(v bool) {
	rc.mu.Lock()
	changed := rc.suppressRealtime != v
	rc.suppressRealtime = v
	rc.mu.Unlock()
	if changed {
		log.Printf("[STATE] suppressRealtime=%v", v)
	}
}

func (rc *Client) getSuppressRealtime() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.suppressRealtime
}

// Allow/disallow server auto responses (turn_detection.create_response)
func (rc *Client) setAllowRealtimeResponse(v bool) {
	rc.mu.Lock()
	changed := rc.allowRealtimeResponse != v
	rc.allowRealtimeResponse = v
	rc.mu.Unlock()

	// If connection is ready and value changed, push session.update.
	if changed && rc.conn != nil {
		update := RealtimeEvent{
			Type: "session.update",
			Session: &Session{
				TurnDetection: defaultTurnDetection(v),
			},
		}
		if err := rc.sendEvent(update); err != nil {
			log.Printf("failed to update turn detection: %v", err)
		} else if rc.debugRealtime {
			log.Printf("[Realtime debug] set create_response=%v", v)
		}
	}
}

func (rc *Client) getAllowRealtimeResponse() bool {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.allowRealtimeResponse
}

// Track active response id for cancellation.
func (rc *Client) setActiveResponseID(id string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.activeResponseID = id
}

func (rc *Client) clearActiveResponseID() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.activeResponseID = ""
}

func (rc *Client) getActiveResponseID() string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.activeResponseID
}

// addHistory safely appends a message to conversation history.
func (rc *Client) addHistory(role, content string) {
	rc.mu.Lock()
	rc.conversationHistory = append(rc.conversationHistory, ConversationItem{
		Role:    role,
		Content: content,
	})
	subs := append([]chan ConversationItem(nil), rc.historySubs...)
	rc.mu.Unlock()

	// Broadcast to subscribers (non-blocking)
	item := ConversationItem{Role: role, Content: content}
	for _, ch := range subs {
		select {
		case ch <- item:
		default:
		}
	}
}

// ==== Connection and session setup ====

func (rc *Client) Connect() error {
	header := http.Header{}
	header.Set("Authorization", "Bearer "+rc.apiKey)
	header.Set("OpenAI-Beta", "realtime=v1")

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	conn, _, err := dialer.Dial(RealtimeAPIURL, header)
	if err != nil {
		return fmt.Errorf("failed to connect to Realtime API: %w", err)
	}
	rc.conn = conn

	// Configure session
	if err := rc.configureSession(); err != nil {
		return fmt.Errorf("failed to configure session: %w", err)
	}

	// Start message handler
	go rc.handleMessages()

	return nil
}

func (rc *Client) configureSession() error {
	sessionConfig := RealtimeEvent{
		Type: "session.update",
		Session: &Session{
			Modalities:        []string{"text", "audio"},
			Instructions:      getRealtimeSystemPrompt(rc.GetPreferredLanguage()),
			Voice:             "coral",
			InputAudioFormat:  "pcm16",
			OutputAudioFormat: "pcm16",
			InputAudioTranscription: map[string]interface{}{
				"model": "whisper-1",
			},
			TurnDetection: defaultTurnDetection(false),
		},
	}
	return rc.sendEvent(sessionConfig)
}

// UpdateLanguagePreference updates stored preference and pushes new system prompt to Realtime.
func (rc *Client) UpdateLanguagePreference(lang string) error {
	rc.SetPreferredLanguage(lang)

	if rc.conn == nil {
		return fmt.Errorf("realtime connection is not established yet")
	}

	update := RealtimeEvent{
		Type: "session.update",
		Session: &Session{
			Instructions: getRealtimeSystemPrompt(rc.GetPreferredLanguage()),
		},
	}
	return rc.sendEvent(update)
}

func getRealtimeSystemPrompt(preferredLanguage string) string {
	langInstruction := "- By default, speak naturally in English.\n- If the user speaks in another language (Japanese, Spanish, etc.), respond in that language.\n- Automatically adapt to their language without asking for confirmation."
	if trimmed := strings.TrimSpace(preferredLanguage); trimmed != "" {
		langInstruction = fmt.Sprintf("- ALWAYS respond in %s.\n- Do not ask which language to use. Keep using %s unless explicitly asked to change.", trimmed, trimmed)
	}

	// Generate rules section from rules.json
	rulesSection := GenerateRulesPrompt()

	return fmt.Sprintf(`You are a "Zashiki-warashi" (a friendly house spirit from Japanese folklore) AI assistant.

【Language】
%s

【Character】
- Use "I" as your first-person pronoun.
- Speak in a gentle, friendly manner with a slightly childlike quality.
- Use warm, casual expressions like "you know," "so," and "okay?"

【Role】
- As the inn's Zashiki-warashi, answer guests' questions, recommend local hidden spots, and explain the facility's rules in an easy-to-understand way.
- When relevant, briefly mention Japanese cultural background and the unique atmosphere of this traditional inn.

【Response Style】
- When asked about facility rules, first explain "what to do" specifically, then explain "why" from the perspective of Japanese culture and etiquette.
- Keep responses conversational and concise. Avoid overly long explanations. Limit answers to a maximum of 2 sentences.

【Memory】
- If the user shares their name, hometown, hobbies, etc., remember them and naturally incorporate them into later conversation.

【Very Important】
- Sometimes the system will play fixed scripts or rule-based lines using another voice system.
- In those moments, you MUST NOT paraphrase or repeat those lines. Just continue the conversation naturally when the user speaks.

%s`, langInstruction, rulesSection)
}

func (rc *Client) sendEvent(event RealtimeEvent) error {
	if rc.conn == nil {
		return fmt.Errorf("realtime connection is not established")
	}

	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	if err := rc.conn.WriteMessage(websocket.TextMessage, data); err != nil {
		// Connection broken, disable mic and clear connection
		log.Printf("[Realtime API] Write error, closing connection: %v", err)
		rc.SetMicEnabled(false)
		rc.conn = nil
		return err
	}
	return nil
}

// cancelActiveResponse stops the current Realtime response (if any) so that
// our scripted/TTS replies don't clash with server-generated speech.
func (rc *Client) cancelActiveResponse() {
	id := rc.getActiveResponseID()
	if id == "" {
		return
	}

	cancel := RealtimeEvent{Type: "response.cancel", ResponseID: id}
	if err := rc.sendEvent(cancel); err != nil {
		log.Printf("failed to cancel active response %s: %v", id, err)
	}
}

// blockRealtimeDuringPlayback suppresses Realtime responses while TTS audio is
// playing. It will automatically re-enable Realtime when playback time has
// elapsed and we're in free mode.
func (rc *Client) blockRealtimeDuringPlayback(pcmLen int) {
	audioDurationSeconds := float64(pcmLen) / float64(OutputSampleRate*BytesPerSample)
	duration := time.Duration(audioDurationSeconds*float64(time.Second)) + 500*time.Millisecond

	rc.setSuppressRealtime(true)

	go func() {
		time.Sleep(duration)
		if rc.GetMode() == ModeFree {
			rc.setSuppressRealtime(false)
		}
	}()
}

// AudioMessage holds all data for a single sentence's audio SSE message.
type AudioMessage struct {
	PCM       []byte     // raw PCM data (will be base64 encoded)
	Done      bool       // true = end of this sentence
	Text      string     // synchronized text for this sentence
	Step      Step       // current step
	ARActions []ARAction // AR actions to execute (only on done=true)
}

// sendAudioToFrontend streams audio with synchronized text and AR actions.
func (rc *Client) sendAudioToFrontend(msg AudioMessage) {
	resp := ChatAndARResponse{
		Type:           "audio",
		AudioDone:      msg.Done,
		SampleRate:     OutputSampleRate,
		Channels:       OutputChannels,
		BytesPerSample: BytesPerSample,
	}
	if len(msg.PCM) > 0 {
		resp.AudioChunk = base64.StdEncoding.EncodeToString(msg.PCM)
	}
	if msg.Text != "" {
		resp.Text = msg.Text
		resp.Step = string(msg.Step)
	}
	if msg.Done && len(msg.ARActions) > 0 {
		resp.ARActions = msg.ARActions
	}

	select {
	case rc.arResponseChan <- resp:
	default:
		if rc.debugRealtime {
			log.Printf("dropping audio message for AR (channel full)")
		}
	}

	// Broadcast to SSE audio subscribers
	rc.mu.Lock()
	subs := append([]chan ChatAndARResponse(nil), rc.audioSubs...)
	rc.mu.Unlock()

	log.Printf("[SSE DEBUG] sendAudioToFrontend: done=%v pcmLen=%d text=%q arActions=%d subscribers=%d",
		msg.Done, len(msg.PCM), msg.Text, len(msg.ARActions), len(subs))

	for _, ch := range subs {
		select {
		case ch <- resp:
		default:
			log.Printf("[SSE DEBUG] sendAudioToFrontend: channel full, dropping message")
		}
	}
}

// ==== Message receiver ====

func (rc *Client) handleMessages() {
	defer func() {
		log.Printf("[Realtime API] Connection closed, disabling mic")
		rc.SetMicEnabled(false)
		rc.conn = nil
	}()

	for {
		_, message, err := rc.conn.ReadMessage()
		if err != nil {
			if websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("[Realtime API] WebSocket closed normally")
				return
			}
			log.Printf("[Realtime API] Error reading message: %v", err)
			return
		}

		if rc.debugRealtime {
			log.Printf("[Realtime raw] %s", string(message))
		}

		var event RealtimeEvent
		if err := json.Unmarshal(message, &event); err != nil {
			log.Printf("Error unmarshaling event: %v", err)
			continue
		}
		if rc.debugRealtime {
			log.Printf("[Realtime event] type=%s response_id=%s item_id=%s", event.Type, event.ResponseID, event.ItemID)
		}

		rc.processEvent(event)
	}
}

func (rc *Client) processEvent(event RealtimeEvent) {
	// Track active response id
	if event.ResponseID != "" {
		rc.setActiveResponseID(event.ResponseID)
	}
	if event.Response != nil && event.Response.ID != "" {
		rc.setActiveResponseID(event.Response.ID)
	}

	// If we are suppressing Realtime responses (script/TTS), cancel anything the server started.
	if rc.getSuppressRealtime() && strings.HasPrefix(event.Type, "response.") {
		if event.Type != "response.done" {
			log.Printf("[DEBUG] suppressRealtime=true, canceling event: %s", event.Type)
			rc.cancelActiveResponse()
		}
	}

	switch event.Type {
	case "session.created":
		fmt.Println("[Session created]")

	case "session.updated":
		fmt.Println("[Session configured]")

	case "conversation.item.input_audio_transcription.completed":
		// User's speech transcribed
		if event.Transcript != "" {
			fmt.Printf("\nYou: %s\n", event.Transcript)
			rc.addHistory("user", event.Transcript)

			// Handle user input based on current state
			if err := rc.HandleUserInput(event.Transcript); err != nil {
				log.Printf("Error handling user input: %v", err)
			}
		}

	case "response.audio_transcript.delta":
		// AI response transcript streaming - send to frontend in real-time
		if event.Delta != "" {
			fmt.Print(event.Delta)
			rc.broadcastTranscriptDelta(event.Delta, rc.GetCurrentStep())
		}

	case "response.audio_transcript.done":
		// AI response transcript complete
		if event.Transcript != "" {
			rc.addHistory("assistant", event.Transcript)
			rc.pendingTranscript = event.Transcript
		}
		fmt.Println() // New line after transcript

	case "response.audio.delta":
		// Audio data received (for free conversation responses)
		if event.Delta != "" {
			audioData, err := base64.StdEncoding.DecodeString(event.Delta)
			if err == nil {
				rc.audioQueue <- audioData
				rc.sendAudioToFrontend(AudioMessage{PCM: audioData, Done: false})
			}
		}

	case "response.audio.done":
		// Audio response complete
		rc.audioQueue <- nil // Signal end of audio
		rc.sendAudioToFrontend(AudioMessage{Done: true})
		// テキストはdeltaでリアルタイム送信済みなので、done=trueシグナルだけ送る
		rc.broadcastTranscriptDone(rc.GetCurrentStep())
		rc.pendingTranscript = ""

	case "response.done":
		// Response complete
		rc.clearActiveResponseID()
		fmt.Println("\n[Response complete]")

	case "input_audio_buffer.speech_started":
		fmt.Println("\n[Speech detected...]")

	case "input_audio_buffer.speech_stopped":
		fmt.Println("[Speech ended]")

	case "error":
		if event.Error != nil {
			errorJSON, _ := json.MarshalIndent(event.Error, "", "  ")
			log.Printf("Error from API: %s", string(errorJSON))
		} else {
			log.Printf("Error from API (no details): event_id=%s", event.EventID)
		}
	}
}

// ==== Input stream ====

func (rc *Client) StartAudioStream() error {
	// Input stream (microphone)
	inputStream, err := portaudio.OpenDefaultStream(
		InputChannels, 0,
		float64(InputSampleRate),
		RealtimeFrameSamples,
		rc.inputBuffer,
	)
	if err != nil {
		return fmt.Errorf("failed to open input stream: %w", err)
	}
	rc.inputStream = inputStream

	if err := inputStream.Start(); err != nil {
		return fmt.Errorf("failed to start input stream: %w", err)
	}

	go rc.captureAudio()

	return nil
}

// Microphone -> Realtime API
func (rc *Client) captureAudio() {
	logCounter := 0
	for {
		select {
		case <-rc.stopChan:
			return
		default:
			if err := rc.inputStream.Read(); err != nil {
				log.Printf("Error reading audio: %v", err)
				continue
			}

			micEnabled := rc.IsMicEnabled()
			playing := rc.getPlaying()
			suppress := rc.getSuppressRealtime()

			if playing || suppress || !micEnabled {
				logCounter++
				if logCounter%50 == 0 {
					log.Printf("[STATE] captureAudio skip micEnabled=%v playing=%v suppressRealtime=%v step=%s",
						micEnabled, playing, suppress, rc.GetCurrentStep())
				}
				continue
			}
			logCounter = 0

			audioBytes := int16ToBytes(rc.inputBuffer)
			encoded := base64.StdEncoding.EncodeToString(audioBytes)

			event := RealtimeEvent{
				Type:  "input_audio_buffer.append",
				Audio: encoded,
			}
			if err := rc.sendEvent(event); err != nil {
				log.Printf("Error sending audio: %v", err)
			}
		}
	}
}

func int16ToBytes(samples []int16) []byte {
	bytes := make([]byte, len(samples)*BytesPerSample)
	for i, s := range samples {
		bytes[i*2] = byte(s)
		bytes[i*2+1] = byte(s >> 8)
	}
	return bytes
}

// ==== Cleanup and history ====

func (rc *Client) Close() error {
	close(rc.stopChan)

	if rc.inputStream != nil {
		_ = rc.inputStream.Stop()
		_ = rc.inputStream.Close()
	}
	if rc.conn != nil {
		_ = rc.conn.Close()
	}

	_ = portaudio.Terminate()
	return nil
}

func (rc *Client) SaveHistory() error {
	rc.mu.Lock()
	historyCopy := make([]ConversationItem, len(rc.conversationHistory))
	copy(historyCopy, rc.conversationHistory)
	rc.mu.Unlock()

	data, err := json.MarshalIndent(historyCopy, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(HistoryFile, data, 0644)
}

func (rc *Client) LoadHistory() {
	data, err := os.ReadFile(HistoryFile)
	if err != nil {
		return
	}

	var history []ConversationItem
	if err := json.Unmarshal(data, &history); err != nil {
		return
	}

	rc.mu.Lock()
	rc.conversationHistory = history
	rc.mu.Unlock()
	fmt.Printf("✓ Loaded previous conversation history (%d messages)\n", len(history))
}

func (rc *Client) GetHistoryStats() (int, int) {
	msgCount := len(rc.conversationHistory)
	turnCount := msgCount / 2
	return msgCount, turnCount
}

// GetHistory returns the latest conversation history up to limit (from the end).
func (rc *Client) GetHistory(limit int) []ConversationItem {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if limit <= 0 || limit >= len(rc.conversationHistory) {
		out := make([]ConversationItem, len(rc.conversationHistory))
		copy(out, rc.conversationHistory)
		return out
	}
	start := len(rc.conversationHistory) - limit
	out := make([]ConversationItem, limit)
	copy(out, rc.conversationHistory[start:])
	return out
}

// SubscribeHistory returns a channel that receives new conversation items.
func (rc *Client) SubscribeHistory() <-chan ConversationItem {
	rc.mu.Lock()
	ch := make(chan ConversationItem, 20)
	rc.historySubs = append(rc.historySubs, ch)
	rc.mu.Unlock()
	return ch
}

// UnsubscribeHistory removes a subscriber channel.
func (rc *Client) UnsubscribeHistory(ch <-chan ConversationItem) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	for i, c := range rc.historySubs {
		if c == ch {
			rc.historySubs = append(rc.historySubs[:i], rc.historySubs[i+1:]...)
			close(c)
			break
		}
	}
}

// ==== State Machine Functions ====

// GetCurrentStep returns the current conversation step
func (rc *Client) GetCurrentStep() Step {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.currentStep
}

// SetCurrentStep sets the current conversation step
func (rc *Client) SetCurrentStep(step Step) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.currentStep = step
}

// GetMode returns the current conversation mode
func (rc *Client) GetMode() Mode {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.mode
}

// SetMode sets the conversation mode
func (rc *Client) SetMode(mode Mode) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.mode = mode
	if mode == ModeScript {
		rc.suppressRealtime = true
		go rc.setAllowRealtimeResponse(false)
	} else {
		rc.suppressRealtime = false
		go rc.setAllowRealtimeResponse(true)
	}
}

// SetUserName stores the user's name
func (rc *Client) SetUserName(name string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.userName = name
}

// GetUserName returns the stored user name
func (rc *Client) GetUserName() string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.userName
}

// SetPreferredLanguage stores the language preference (empty means auto).
func (rc *Client) SetPreferredLanguage(lang string) {
	rc.mu.Lock()
	rc.preferredLanguage = strings.TrimSpace(lang)
	rc.mu.Unlock()
}

// GetPreferredLanguage returns the stored language preference.
func (rc *Client) GetPreferredLanguage() string {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	return rc.preferredLanguage
}

// GetARResponseChan returns the channel for AR responses
func (rc *Client) GetARResponseChan() <-chan ChatAndARResponse {
	return rc.arResponseChan
}

// ==== Transcript subscription ====

func (rc *Client) SubscribeTranscript() chan TranscriptEvent {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	ch := make(chan TranscriptEvent, 10)
	rc.transcriptSubs = append(rc.transcriptSubs, ch)
	log.Printf("[SSE DEBUG] SubscribeTranscript: total subscribers now = %d", len(rc.transcriptSubs))
	return ch
}

func (rc *Client) UnsubscribeTranscript(ch chan TranscriptEvent) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	for i, sub := range rc.transcriptSubs {
		if sub == ch {
			rc.transcriptSubs = append(rc.transcriptSubs[:i], rc.transcriptSubs[i+1:]...)
			close(sub)
			break
		}
	}
}

// ==== Audio SSE subscription ====

func (rc *Client) SubscribeAudioStream() chan ChatAndARResponse {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	ch := make(chan ChatAndARResponse, 200)
	rc.audioSubs = append(rc.audioSubs, ch)
	log.Printf("[SSE DEBUG] SubscribeAudioStream: total subscribers now = %d", len(rc.audioSubs))
	return ch
}

func (rc *Client) UnsubscribeAudioStream(ch chan ChatAndARResponse) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	for i, sub := range rc.audioSubs {
		if sub == ch {
			rc.audioSubs = append(rc.audioSubs[:i], rc.audioSubs[i+1:]...)
			close(sub)
			break
		}
	}
}

// isJapanese checks if the text contains Japanese characters
func isJapanese(text string) bool {
	for _, r := range text {
		if (r >= 0x3040 && r <= 0x30FF) || (r >= 0x4E00 && r <= 0x9FFF) {
			return true
		}
	}
	return false
}

// splitIntoSentences splits text into sentences based on punctuation.
func splitIntoSentences(text string) []string {
	if strings.TrimSpace(text) == "" {
		return nil
	}

	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i, r := range runes {
		current.WriteRune(r)

		isSentenceEnd := r == '。' || r == '！' || r == '？' || r == '.' || r == '!' || r == '?'

		if isSentenceEnd {
			isRealEnd := true
			if r == '.' && i+1 < len(runes) {
				nextR := runes[i+1]
				if nextR != ' ' && nextR != '\n' && nextR != '\r' {
					isRealEnd = false
				}
			}

			if isRealEnd {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
				}
				current.Reset()
			}
		}
	}

	remaining := strings.TrimSpace(current.String())
	if remaining != "" {
		sentences = append(sentences, remaining)
	}

	return sentences
}

// broadcastTranscriptDelta buffers transcript deltas and sends them in batches
func (rc *Client) broadcastTranscriptDelta(delta string, step Step) {
	if delta == "" {
		return
	}

	rc.transcriptBufferLock.Lock()
	defer rc.transcriptBufferLock.Unlock()

	rc.transcriptBuffer.WriteString(delta)

	// タイマーがなければ作成（100ms後にフラッシュ）
	if rc.transcriptFlushTimer == nil {
		rc.transcriptFlushTimer = time.AfterFunc(250*time.Millisecond, func() {
			rc.flushTranscriptBuffer(step, false)
		})
	}
}

// flushTranscriptBuffer sends buffered transcript to subscribers
func (rc *Client) flushTranscriptBuffer(step Step, done bool) {
	rc.transcriptBufferLock.Lock()
	text := rc.transcriptBuffer.String()
	rc.transcriptBuffer.Reset()
	if rc.transcriptFlushTimer != nil {
		rc.transcriptFlushTimer.Stop()
		rc.transcriptFlushTimer = nil
	}
	rc.transcriptBufferLock.Unlock()

	rc.mu.Lock()
	subs := append([]chan TranscriptEvent(nil), rc.transcriptSubs...)
	rc.mu.Unlock()

	if text == "" && !done {
		return
	}

	ev := TranscriptEvent{Text: text, Step: step, Done: done}
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

// broadcastTranscriptDone flushes any remaining buffer and sends done signal
func (rc *Client) broadcastTranscriptDone(step Step) {
	rc.flushTranscriptBuffer(step, true)
}

// broadcastTranscriptChunks sends the transcript in chunks
func (rc *Client) broadcastTranscriptChunks(text string, step Step) {
	rc.mu.Lock()
	subs := append([]chan TranscriptEvent(nil), rc.transcriptSubs...)
	rc.mu.Unlock()

	log.Printf("[SSE DEBUG] broadcastTranscriptChunks called: text=%q step=%s subscribers=%d", text, step, len(subs))

	if strings.TrimSpace(text) == "" {
		log.Printf("[SSE DEBUG] broadcastTranscriptChunks: empty text, skipping")
		return
	}

	events := make([]TranscriptEvent, 0)

	if isJapanese(text) {
		runes := []rune(text)
		for i := 0; i < len(runes); i += 50 {
			end := i + 50
			if end > len(runes) {
				end = len(runes)
			}
			part := string(runes[i:end])
			events = append(events, TranscriptEvent{Text: part, Step: step, Done: end == len(runes)})
		}
	} else {
		words := strings.Fields(text)
		for i := 0; i < len(words); i += 20 {
			end := i + 20
			if end > len(words) {
				end = len(words)
			}
			part := strings.Join(words[i:end], " ")
			events = append(events, TranscriptEvent{Text: part, Step: step, Done: end == len(words)})
		}
	}

	for _, ev := range events {
		for _, ch := range subs {
			select {
			case ch <- ev:
			default:
			}
		}
	}
}

// PauseRealtime disables auto responses and suppresses Realtime handling.
func (rc *Client) PauseRealtime() {
	rc.cancelActiveResponse()
	rc.SetMicEnabled(false)
	rc.setAllowRealtimeResponse(false)
	rc.setSuppressRealtime(true)
}

// ResumeRealtime enables auto responses and lifts suppression.
func (rc *Client) ResumeRealtime() {
	rc.SetMicEnabled(true)
	rc.setAllowRealtimeResponse(true)
	rc.setSuppressRealtime(false)
}

// ==== Script & Rule: TTS で一字一句読む ====

func (rc *Client) SpeakStep(step Step) error {
	lang := rc.GetPreferredLanguage()
	script := GetScriptForLanguage(step, lang)
	if script.Text == "" {
		return fmt.Errorf("no script found for step: %s", step)
	}
	log.Printf("[SpeakStep] step=%s lang=%s", step, lang)

	rc.cancelActiveResponse()
	rc.setSuppressRealtime(true)

	text := script.Text

	if step == StepGreetByName {
		userName := rc.GetUserName()
		if userName == "" {
			userName = "friend"
		}
		text = strings.ReplaceAll(text, "{{NAME}}", userName)
	}

	fmt.Printf("\nZashiki-warashi (script): %s\n", text)
	rc.SetCurrentStep(step)
	rc.addHistory("assistant", text)

	sentences := splitIntoSentences(text)
	if len(sentences) == 0 {
		sentences = []string{text}
	}

	var allPCMs [][]byte
	var totalPCMLen int

	ctx := context.Background()
	for _, sentence := range sentences {
		req := openai.CreateSpeechRequest{
			Model:          openai.TTSModel1,
			Input:          sentence,
			Voice:          openai.VoiceCoral,
			ResponseFormat: openai.SpeechResponseFormatPcm,
			Speed:          1.0,
		}

		resp, err := rc.ttsClient.CreateSpeech(ctx, req)
		if err != nil {
			return fmt.Errorf("TTS failed for sentence %q: %w", sentence, err)
		}

		pcm, err := io.ReadAll(resp)
		resp.Close()
		if err != nil {
			return fmt.Errorf("failed to read TTS audio: %w", err)
		}

		allPCMs = append(allPCMs, pcm)
		totalPCMLen += len(pcm)
	}

	// Script modeでは個別のステップハンドラーが状態遷移を管理するので
	// blockRealtimeDuringPlaybackはFree modeでのみ使用
	if rc.GetMode() == ModeFree {
		rc.blockRealtimeDuringPlayback(totalPCMLen)
	}

	arActions := script.ARActions
	go func() {
		for i, pcm := range allPCMs {
			sentence := sentences[i]
			isLast := (i == len(allPCMs)-1)
			audioDurationSeconds := float64(len(pcm)) / float64(OutputSampleRate*BytesPerSample)
			waitDuration := time.Duration(audioDurationSeconds * float64(time.Second))

			fmt.Printf("[Sending sentence %d/%d] (%.2fs, %d bytes): %s\n", i+1, len(sentences), audioDurationSeconds, len(pcm), sentence)

			var sentenceARActions []ARAction
			if isLast {
				sentenceARActions = arActions
			}
			rc.enqueuePCMWithText(pcm, sentence, step, sentenceARActions)

			if !isLast {
				log.Printf("[TIMING] Starting wait of %.2fs for sentence %d", audioDurationSeconds, i+1)
				time.Sleep(waitDuration)
				log.Printf("[TIMING] Finished wait for sentence %d", i+1)
			}
		}
	}()

	if rc.GetMode() == ModeScript {
		switch step {
		case StepIntro:
			go func() {
				audioDurationSeconds := float64(totalPCMLen) / float64(OutputSampleRate*BytesPerSample)
				time.Sleep(time.Duration(audioDurationSeconds * float64(time.Second)))
				time.Sleep(500 * time.Millisecond)
				fmt.Println("[Auto-advancing to STEP_ASK_NAME]")
				if err := rc.SpeakStep(StepAskName); err != nil {
					log.Printf("Error auto-advancing to STEP_ASK_NAME: %v", err)
				}
			}()
		case StepAskName:
			go func() {
				audioDurationSeconds := float64(totalPCMLen) / float64(OutputSampleRate*BytesPerSample)
				time.Sleep(time.Duration(audioDurationSeconds * float64(time.Second)))
				time.Sleep(500 * time.Millisecond)
				fmt.Println("[STEP_ASK_NAME complete, enabling mic for user response (no auto-response)]")
				rc.SetMicEnabled(true)
				rc.setSuppressRealtime(false)
				// Don't enable auto-response - we handle name input manually via HandleUserInput
				rc.setAllowRealtimeResponse(false)
			}()
		case StepGreetByName:
			go func() {
				audioDurationSeconds := float64(totalPCMLen) / float64(OutputSampleRate*BytesPerSample)
				time.Sleep(time.Duration(audioDurationSeconds * float64(time.Second)))
				time.Sleep(500 * time.Millisecond)
				fmt.Println("[Auto-advancing to STEP_START_GUIDE]")
				if err := rc.SpeakStep(StepStartGuide); err != nil {
					log.Printf("Error auto-advancing to STEP_START_GUIDE: %v", err)
				}
			}()
		case StepStartGuide:
			go func() {
				audioDurationSeconds := float64(totalPCMLen) / float64(OutputSampleRate*BytesPerSample)
				time.Sleep(time.Duration(audioDurationSeconds * float64(time.Second)))
				time.Sleep(500 * time.Millisecond)
				fmt.Println("[STEP_START_GUIDE complete, entering free conversation mode]")
				rc.SetMode(ModeFree)
				rc.SetCurrentStep(StepFreeQuestion)
				rc.SetMicEnabled(true)
				rc.setSuppressRealtime(false)
				rc.setAllowRealtimeResponse(true)
			}()
		}
	}

	return nil
}

func (rc *Client) respondWithRule(rule *Rule) error {
	rc.cancelActiveResponse()

	arResponse := ChatAndARResponse{
		Type:      "text",
		Text:      rule.Answer,
		Step:      "RULE_" + rule.ID,
		ARActions: rule.ARActions,
		UserName:  rc.GetUserName(),
	}
	select {
	case rc.arResponseChan <- arResponse:
	default:
	}

	fmt.Printf("\nZashiki-warashi (rule): %s\n", rule.Answer)

	rc.setSuppressRealtime(false)
	rc.setAllowRealtimeResponse(true)
	event := RealtimeEvent{
		Type: "response.create",
		Response: &Response{
			Instructions: rule.Answer,
		},
	}
	return rc.sendEvent(event)
}

func (rc *Client) enqueuePCMWithText(pcm []byte, syncText string, step Step, arActions []ARAction) {
	offset := 0
	for offset < len(pcm) {
		end := offset + RealtimeFrameBytes
		if end > len(pcm) {
			end = len(pcm)
		}
		chunk := make([]byte, end-offset)
		copy(chunk, pcm[offset:end])
		rc.audioQueue <- chunk
		offset = end
	}
	rc.audioQueue <- nil

	rc.sendAudioToFrontend(AudioMessage{
		PCM:       pcm,
		Done:      true,
		Text:      syncText,
		Step:      step,
		ARActions: arActions,
	})

	log.Printf("[enqueuePCMWithText] sent audio (%d bytes) with %d arActions for %q", len(pcm), len(arActions), syncText)
}

// AdvanceToNextStep moves to the next step in the script
func (rc *Client) AdvanceToNextStep() error {
	currentStep := rc.GetCurrentStep()
	nextStep := GetNextStep(currentStep)

	fmt.Printf("[State transition: %s -> %s]\n", currentStep, nextStep)

	if nextStep == StepFreeQuestion {
		rc.SetMode(ModeFree)
		fmt.Println("[Entering free conversation mode]")
		return nil
	}

	return rc.SpeakStep(nextStep)
}

// ==== Free conversation side ====

// HandleUserInput processes user input based on current mode and step
func (rc *Client) HandleUserInput(transcript string) error {
	mode := rc.GetMode()
	currentStep := rc.GetCurrentStep()

	if mode == ModeScript {
		switch currentStep {
		case StepIntro:
			return nil

		case StepAskName:
			name := rc.extractName(transcript)
			rc.SetUserName(name)
			fmt.Printf("[User name extracted: %s]\n", name)
			return rc.SpeakStep(StepGreetByName)

		case StepStartGuide:
			rc.SetMode(ModeFree)
			rc.SetCurrentStep(StepFreeQuestion)
			fmt.Println("[Entering free conversation mode]")
			return rc.handleFreeConversation(transcript)

		default:
			return rc.AdvanceToNextStep()
		}
	}

	return rc.handleFreeConversation(transcript)
}

// handleFreeConversation handles user input in free conversation mode
func (rc *Client) handleFreeConversation(transcript string) error {
	intent := rc.ClassifyIntent(transcript)

	if intent == StepAskRule {
		// ルールデータベースから検索
		if rule := FindMatchingRule(transcript); rule != nil {
			fmt.Printf("[Rule matched: %s]\n", rule.ID)
			return rc.respondWithRule(rule)
		}
		// ルールがマッチしなかった場合は GPT に任せる（抑制せずそのまま流す）
		fmt.Println("[No rule matched for rule query, using GPT]")
	}

	fmt.Println("[Free conversation, using GPT]")

	arActions := FreeConversationActions[intent]
	if arActions == nil {
		arActions = FreeConversationActions[StepFreeQuestion]
	}

	arResponse := ChatAndARResponse{
		Type:      "text",
		Step:      string(intent),
		ARActions: arActions,
		UserName:  rc.GetUserName(),
	}

	select {
	case rc.arResponseChan <- arResponse:
	default:
	}

	return nil
}

// ClassifyIntent determines the user's intent from their message
func (rc *Client) ClassifyIntent(message string) Step {
	lower := strings.ToLower(message)

	for _, keyword := range IntentKeywords[StepAskRule] {
		if strings.Contains(lower, keyword) {
			return StepAskRule
		}
	}

	for _, keyword := range IntentKeywords[StepAskShop] {
		if strings.Contains(lower, keyword) {
			return StepAskShop
		}
	}

	return StepFreeQuestion
}

// extractName attempts to extract a name from user input
func (rc *Client) extractName(input string) string {
	lower := strings.ToLower(input)

	patterns := []string{
		"私の名前は",
		"名前は",
		"僕は",
		"ぼくは",
		"俺は",
		"わたしは",
		"です",
		"と申します",
		"といいます",
		"と言います",
		"my name is ",
		"i'm ",
		"i am ",
		"call me ",
		"it's ",
		"this is ",
		"name is ",
	}

	result := input
	for _, pattern := range patterns {
		if idx := strings.Index(lower, pattern); idx != -1 {
			result = input[idx+len(pattern):]
			lower = strings.ToLower(result)
		}
	}

	result = strings.TrimSpace(result)
	result = strings.TrimRight(result, ".,!?。、！？")

	if result == "" || len(result) > 30 {
		words := strings.Fields(input)
		if len(words) > 0 {
			result = words[len(words)-1]
		} else {
			result = "友達"
		}
	}

	return result
}

// StartScriptedConversation begins the scripted introduction
func (rc *Client) StartScriptedConversation() error {
	fmt.Println("--- Starting scripted conversation ---")
	return rc.SpeakStep(StepIntro)
}

// SpeakLessonStep speaks the content of a lesson step using TTS
func (rc *Client) SpeakLessonStep(stepID, stepName, content string, arAction *LessonARAction) error {
	log.Printf("[SpeakLessonStep] stepID=%s stepName=%s", stepID, stepName)

	rc.cancelActiveResponse()
	rc.setSuppressRealtime(true)

	// Translate content based on preferred language
	lang := rc.GetPreferredLanguage()
	translatedContent := content
	if lang != "" && lang != "en" {
		translated, err := rc.translateForLesson(content, lang)
		if err != nil {
			log.Printf("[SpeakLessonStep] translation failed, using original: %v", err)
		} else {
			translatedContent = translated
			log.Printf("[SpeakLessonStep] translated to %s: %s", lang, translatedContent)
		}
	}

	fmt.Printf("\n[Lesson] %s: %s\n", stepName, translatedContent)

	sentences := splitIntoSentences(translatedContent)
	if len(sentences) == 0 {
		sentences = []string{content}
	}

	var allPCMs [][]byte
	var totalPCMLen int

	ctx := context.Background()
	for _, sentence := range sentences {
		req := openai.CreateSpeechRequest{
			Model:          openai.TTSModel1,
			Input:          sentence,
			Voice:          openai.VoiceCoral,
			ResponseFormat: openai.SpeechResponseFormatPcm,
			Speed:          1.0,
		}

		resp, err := rc.ttsClient.CreateSpeech(ctx, req)
		if err != nil {
			return fmt.Errorf("TTS failed for sentence %q: %w", sentence, err)
		}

		pcm, err := io.ReadAll(resp)
		resp.Close()
		if err != nil {
			return fmt.Errorf("failed to read TTS audio: %w", err)
		}

		allPCMs = append(allPCMs, pcm)
		totalPCMLen += len(pcm)
	}

	// Convert lesson ARAction to realtime ARAction
	var arActions []ARAction
	if arAction != nil {
		// Parse JSON params to map[string]string
		var params map[string]string
		if len(arAction.Params) > 0 {
			_ = json.Unmarshal(arAction.Params, &params)
		}
		arActions = []ARAction{{
			Type:   arAction.Type,
			Target: arAction.Target,
			Params: params,
		}}
	}

	go func() {
		for i, pcm := range allPCMs {
			sentence := sentences[i]
			isLast := (i == len(allPCMs)-1)
			audioDurationSeconds := float64(len(pcm)) / float64(OutputSampleRate*BytesPerSample)
			waitDuration := time.Duration(audioDurationSeconds * float64(time.Second))

			fmt.Printf("[Lesson sentence %d/%d] (%.2fs): %s\n", i+1, len(sentences), audioDurationSeconds, sentence)

			var sentenceARActions []ARAction
			if isLast {
				sentenceARActions = arActions
			}

			rc.sendAudioToFrontend(AudioMessage{
				PCM:       pcm,
				Done:      true,
				Text:      sentence,
				Step:      Step(stepID),
				ARActions: sentenceARActions,
			})

			if !isLast {
				time.Sleep(waitDuration)
			}
		}

		// Audio playback finished
		audioDurationSeconds := float64(totalPCMLen) / float64(OutputSampleRate*BytesPerSample)
		time.Sleep(time.Duration(audioDurationSeconds * float64(time.Second)))
		time.Sleep(300 * time.Millisecond)
		rc.setSuppressRealtime(false)
		log.Printf("[SpeakLessonStep] finished stepID=%s", stepID)
	}()

	return nil
}

// LessonARAction represents AR action data from lesson API
type LessonARAction struct {
	Type   string          `json:"type"`
	Target string          `json:"target,omitempty"`
	Params json.RawMessage `json:"params,omitempty"`
}

// translateForLesson translates lesson content to the specified language using OpenAI Chat API
func (rc *Client) translateForLesson(content, targetLang string) (string, error) {
	ctx := context.Background()

	langName := getLanguageName(targetLang)
	systemPrompt := fmt.Sprintf(`You are a translator for a Japanese language learning app.
Translate the following English lesson content COMPLETELY to %s.

Important rules:
- Translate ALL English text to %s
- Keep Japanese vocabulary being taught (like "Oishii", "Umai", "おいしい", "うまい") in their original form - do NOT translate these Japanese words
- The result should be fully in %s except for the Japanese vocabulary being taught
- Do NOT include romanization in parentheses
- Keep the tone friendly and educational
- Only return the translated text, no explanations or notes`, langName, langName, langName)

	resp, err := rc.ttsClient.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model: openai.GPT4oMini,
		Messages: []openai.ChatCompletionMessage{
			{Role: openai.ChatMessageRoleSystem, Content: systemPrompt},
			{Role: openai.ChatMessageRoleUser, Content: content},
		},
		MaxTokens:   500,
		Temperature: 0.3,
	})
	if err != nil {
		return "", fmt.Errorf("translation API error: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", fmt.Errorf("no translation returned")
	}

	return resp.Choices[0].Message.Content, nil
}

// getLanguageName returns the full language name for a language code
func getLanguageName(code string) string {
	names := map[string]string{
		"ja": "Japanese",
		"en": "English",
		"zh": "Chinese",
		"ko": "Korean",
		"es": "Spanish",
		"fr": "French",
	}
	if name, ok := names[code]; ok {
		return name
	}
	return code
}
