package realtime

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/generative-ai-go/genai"
	"github.com/gordonklaus/portaudio"
	openai "github.com/sashabaranov/go-openai"
	"google.golang.org/api/option"
)

const (
	// Gemini audio format (same as OpenAI for compatibility)
	GeminiInputSampleRate  = 24000
	GeminiOutputSampleRate = 24000

	// 24kHz × 200ms = 4,800 samples
	GeminiFrameSamples = 4800
	GeminiFrameBytes   = GeminiFrameSamples * BytesPerSample

	// VAD settings
	SilenceThreshold    = 500  // Amplitude threshold for silence detection
	SilenceDurationMs   = 1000 // How long silence before processing (ms)
	MinSpeechDurationMs = 500  // Minimum speech duration to process
)

// GeminiClient handles voice chat using Gemini 1.5 + Whisper + TTS
type GeminiClient struct {
	geminiClient *genai.Client
	chatSession  *genai.ChatSession
	ttsClient    *openai.Client
	inputStream  *portaudio.Stream

	inputBuffer   []int16
	audioQueue    chan []byte
	recordedAudio []int16 // Buffer for recording speech
	isRecording   bool
	silenceStart  time.Time
	speechStart   time.Time

	transcriptChan chan string
	responseChan   chan string

	stopChan chan struct{}

	// State management
	mu                    sync.Mutex
	isPlaying             bool
	micEnabled            bool
	suppressRealtime      bool
	preferredLanguage     string
	allowRealtimeResponse bool

	debugGemini bool

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

	// Transcript streaming
	transcriptSubs    []chan TranscriptEvent
	pendingTranscript string

	// Transcript buffer for batched sending
	transcriptBuffer     strings.Builder
	transcriptBufferLock sync.Mutex
	transcriptFlushTimer *time.Timer
}

// NewGeminiClient creates a new Gemini 1.5 + TTS client
func NewGeminiClient() (*GeminiClient, error) {
	geminiKey := os.Getenv("GEMINI_API_KEY")
	if geminiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable not set")
	}

	openaiKey := os.Getenv("OPENAI_API_KEY")
	if openaiKey == "" {
		return nil, fmt.Errorf("OPENAI_API_KEY environment variable not set (needed for TTS/Whisper)")
	}

	ctx := context.Background()
	geminiClient, err := genai.NewClient(ctx, option.WithAPIKey(geminiKey))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gemini client: %w", err)
	}

	ttsClient := openai.NewClient(openaiKey)

	gc := &GeminiClient{
		geminiClient:   geminiClient,
		ttsClient:      ttsClient,
		inputBuffer:    make([]int16, GeminiFrameSamples),
		audioQueue:     make(chan []byte, 100),
		recordedAudio:  make([]int16, 0),
		transcriptChan: make(chan string, 10),
		responseChan:   make(chan string, 10),
		stopChan:       make(chan struct{}),
		arResponseChan: make(chan ChatAndARResponse, 10),
		mode:           ModeScript,
		currentStep:    StepIntro,
		debugGemini:    os.Getenv("DEBUG_GEMINI") == "true",
	}

	return gc, nil
}

// Connect initializes the Gemini chat session
func (gc *GeminiClient) Connect() error {
	model := gc.geminiClient.GenerativeModel("gemini-1.5-flash")

	model.SystemInstruction = &genai.Content{
		Parts: []genai.Part{
			genai.Text(gc.getSystemPrompt()),
		},
	}

	// Configure generation settings
	model.SetTemperature(0.7)
	model.SetTopP(0.9)
	model.SetMaxOutputTokens(256) // Keep responses short

	gc.chatSession = model.StartChat()

	log.Printf("[Gemini] Connected to Gemini 1.5 Flash")
	return nil
}

func (gc *GeminiClient) getSystemPrompt() string {
	lang := gc.GetPreferredLanguage()
	langInstruction := ""
	if lang != "" {
		langName := getLanguageName(lang)
		langInstruction = fmt.Sprintf("\n\n【Language】\nThe user prefers %s. Always respond in %s.", langName, langName)
	}

	return fmt.Sprintf(`You are Zashiki-warashi (座敷童子), a friendly spirit from Japanese folklore who lives in this traditional Japanese inn (ryokan).

【Character】
- You are playful, warm, and slightly mischievous, but always kind
- You use casual, friendly Japanese (or the user's preferred language)
- You know everything about this ryokan and Japanese culture
- Keep responses conversational and concise. Limit answers to 2 sentences maximum.

【Memory】
- If the user shares their name, hometown, hobbies, etc., remember them and incorporate them naturally.

%s`, langInstruction)
}

// ProcessSpeech converts speech to text, gets Gemini response, and speaks it
func (gc *GeminiClient) ProcessSpeech(audioData []int16) error {
	if len(audioData) < GeminiInputSampleRate/2 { // Less than 0.5 seconds
		log.Printf("[Gemini] Audio too short, skipping")
		return nil
	}

	// 1. Convert audio to text using Whisper
	transcript, err := gc.speechToText(audioData)
	if err != nil {
		return fmt.Errorf("speech to text failed: %w", err)
	}
	if strings.TrimSpace(transcript) == "" {
		log.Printf("[Gemini] Empty transcript, skipping")
		return nil
	}

	fmt.Printf("\nYou: %s\n", transcript)
	gc.addHistory("user", transcript)

	// 2. Get response from Gemini
	response, err := gc.getGeminiResponse(transcript)
	if err != nil {
		return fmt.Errorf("gemini response failed: %w", err)
	}

	fmt.Printf("Zashiki-warashi: %s\n", response)
	gc.addHistory("assistant", response)

	// 3. Broadcast text to frontend
	gc.broadcastTranscriptDelta(response, gc.GetCurrentStep())
	gc.broadcastTranscriptDone(gc.GetCurrentStep())

	// 4. Convert response to speech using TTS
	if err := gc.textToSpeech(response); err != nil {
		return fmt.Errorf("text to speech failed: %w", err)
	}

	return nil
}

// speechToText converts audio to text using OpenAI Whisper
func (gc *GeminiClient) speechToText(audioData []int16) (string, error) {
	ctx := context.Background()

	// Convert int16 samples to bytes (PCM16 little-endian)
	audioBytes := int16ToBytes(audioData)

	// Create a temporary WAV file in memory
	wavData := createWAVHeader(audioBytes, GeminiInputSampleRate, 1, 16)
	wavData = append(wavData, audioBytes...)

	// Use Whisper API
	resp, err := gc.ttsClient.CreateTranscription(ctx, openai.AudioRequest{
		Model:    openai.Whisper1,
		Reader:   strings.NewReader(string(wavData)),
		FilePath: "audio.wav",
		Language: gc.getWhisperLanguage(),
	})
	if err != nil {
		return "", err
	}

	return resp.Text, nil
}

func (gc *GeminiClient) getWhisperLanguage() string {
	lang := gc.GetPreferredLanguage()
	switch lang {
	case "ja":
		return "ja"
	case "zh":
		return "zh"
	case "ko":
		return "ko"
	case "es":
		return "es"
	case "fr":
		return "fr"
	default:
		return "en"
	}
}

// getGeminiResponse gets a response from Gemini
func (gc *GeminiClient) getGeminiResponse(userMessage string) (string, error) {
	ctx := context.Background()

	resp, err := gc.chatSession.SendMessage(ctx, genai.Text(userMessage))
	if err != nil {
		return "", err
	}

	if len(resp.Candidates) == 0 || len(resp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("no response from Gemini")
	}

	// Extract text from response
	var result strings.Builder
	for _, part := range resp.Candidates[0].Content.Parts {
		if text, ok := part.(genai.Text); ok {
			result.WriteString(string(text))
		}
	}

	return result.String(), nil
}

// textToSpeech converts text to speech and sends to frontend
func (gc *GeminiClient) textToSpeech(text string) error {
	ctx := context.Background()

	gc.SetPlaying(true)
	defer gc.SetPlaying(false)

	// Split into sentences for smoother streaming
	sentences := splitIntoSentences(text)
	if len(sentences) == 0 {
		sentences = []string{text}
	}

	for i, sentence := range sentences {
		req := openai.CreateSpeechRequest{
			Model:          openai.TTSModel1,
			Input:          sentence,
			Voice:          openai.VoiceCoral,
			ResponseFormat: openai.SpeechResponseFormatPcm,
			Speed:          1.0,
		}

		resp, err := gc.ttsClient.CreateSpeech(ctx, req)
		if err != nil {
			return fmt.Errorf("TTS failed: %w", err)
		}

		pcm, err := io.ReadAll(resp)
		resp.Close()
		if err != nil {
			return fmt.Errorf("failed to read TTS audio: %w", err)
		}

		isLast := (i == len(sentences)-1)
		gc.sendAudioToFrontend(AudioMessage{
			PCM:  pcm,
			Done: isLast,
			Text: sentence,
			Step: gc.GetCurrentStep(),
		})

		// Wait for audio duration before sending next chunk
		if !isLast {
			audioDuration := float64(len(pcm)) / float64(GeminiOutputSampleRate*BytesPerSample)
			time.Sleep(time.Duration(audioDuration * float64(time.Second)))
		}
	}

	return nil
}

// StartAudioStream starts capturing audio from microphone with VAD
func (gc *GeminiClient) StartAudioStream() error {
	inputStream, err := portaudio.OpenDefaultStream(
		InputChannels, 0,
		float64(GeminiInputSampleRate),
		GeminiFrameSamples,
		gc.inputBuffer,
	)
	if err != nil {
		return fmt.Errorf("failed to open input stream: %w", err)
	}
	gc.inputStream = inputStream

	if err := inputStream.Start(); err != nil {
		return fmt.Errorf("failed to start input stream: %w", err)
	}

	go gc.captureAudioWithVAD()

	log.Printf("[Gemini] Audio stream started with VAD (24kHz)")
	return nil
}

// captureAudioWithVAD captures audio and detects speech using VAD
func (gc *GeminiClient) captureAudioWithVAD() {
	for {
		select {
		case <-gc.stopChan:
			return
		default:
		}

		if err := gc.inputStream.Read(); err != nil {
			log.Printf("[Gemini] Error reading audio: %v", err)
			continue
		}

		micEnabled := gc.IsMicEnabled()
		playing := gc.IsPlaying()
		suppress := gc.getSuppressRealtime()

		if playing || suppress || !micEnabled {
			continue
		}

		// Calculate audio amplitude
		amplitude := gc.calculateAmplitude(gc.inputBuffer)

		if amplitude > SilenceThreshold {
			// Speech detected
			if !gc.isRecording {
				gc.isRecording = true
				gc.speechStart = time.Now()
				gc.recordedAudio = make([]int16, 0)
				fmt.Println("\n[Speech detected...]")
			}
			gc.silenceStart = time.Time{} // Reset silence timer
			gc.recordedAudio = append(gc.recordedAudio, gc.inputBuffer...)
		} else if gc.isRecording {
			// Silence during recording
			gc.recordedAudio = append(gc.recordedAudio, gc.inputBuffer...)

			if gc.silenceStart.IsZero() {
				gc.silenceStart = time.Now()
			} else if time.Since(gc.silenceStart) > time.Duration(SilenceDurationMs)*time.Millisecond {
				// Silence long enough - process the audio
				fmt.Println("[Speech ended]")
				gc.isRecording = false

				speechDuration := time.Since(gc.speechStart)
				if speechDuration > time.Duration(MinSpeechDurationMs)*time.Millisecond {
					// Process in goroutine to not block capture
					audioData := make([]int16, len(gc.recordedAudio))
					copy(audioData, gc.recordedAudio)
					go func() {
						if err := gc.ProcessSpeech(audioData); err != nil {
							log.Printf("[Gemini] Error processing speech: %v", err)
						}
					}()
				}
				gc.recordedAudio = make([]int16, 0)
			}
		}
	}
}

func (gc *GeminiClient) calculateAmplitude(samples []int16) int {
	var sum int64
	for _, s := range samples {
		if s < 0 {
			sum += int64(-s)
		} else {
			sum += int64(s)
		}
	}
	return int(sum / int64(len(samples)))
}

// Close closes the Gemini connection
func (gc *GeminiClient) Close() error {
	close(gc.stopChan)

	if gc.inputStream != nil {
		gc.inputStream.Stop()
		gc.inputStream.Close()
	}

	if gc.geminiClient != nil {
		gc.geminiClient.Close()
	}

	return nil
}

// Helper to create WAV header
func createWAVHeader(audioData []byte, sampleRate, channels, bitsPerSample int) []byte {
	dataSize := len(audioData)
	fileSize := 36 + dataSize

	header := make([]byte, 44)
	copy(header[0:4], "RIFF")
	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)
	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	header[16] = 16 // Subchunk1Size
	header[20] = 1  // AudioFormat (PCM)
	header[22] = byte(channels)
	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)
	byteRate := sampleRate * channels * bitsPerSample / 8
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)
	header[32] = byte(channels * bitsPerSample / 8) // BlockAlign
	header[34] = byte(bitsPerSample)
	copy(header[36:40], "data")
	header[40] = byte(dataSize)
	header[41] = byte(dataSize >> 8)
	header[42] = byte(dataSize >> 16)
	header[43] = byte(dataSize >> 24)

	return header
}

// addHistory adds a message to conversation history
func (gc *GeminiClient) addHistory(role, content string) {
	gc.mu.Lock()
	gc.conversationHistory = append(gc.conversationHistory, ConversationItem{
		Role:    role,
		Content: content,
	})
	gc.mu.Unlock()
}

// State management methods

func (gc *GeminiClient) SetMicEnabled(enabled bool) {
	gc.mu.Lock()
	changed := gc.micEnabled != enabled
	gc.micEnabled = enabled
	gc.mu.Unlock()
	if changed {
		log.Printf("[STATE] micEnabled=%v", enabled)
	}
}

func (gc *GeminiClient) IsMicEnabled() bool {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.micEnabled
}

func (gc *GeminiClient) SetPlaying(playing bool) {
	gc.mu.Lock()
	gc.isPlaying = playing
	gc.mu.Unlock()
}

func (gc *GeminiClient) IsPlaying() bool {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.isPlaying
}

func (gc *GeminiClient) setSuppressRealtime(v bool) {
	gc.mu.Lock()
	changed := gc.suppressRealtime != v
	gc.suppressRealtime = v
	gc.mu.Unlock()
	if changed {
		log.Printf("[STATE] suppressRealtime=%v", v)
	}
}

func (gc *GeminiClient) getSuppressRealtime() bool {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.suppressRealtime
}

func (gc *GeminiClient) GetCurrentStep() Step {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.currentStep
}

func (gc *GeminiClient) SetCurrentStep(step Step) {
	gc.mu.Lock()
	gc.currentStep = step
	gc.mu.Unlock()
	log.Printf("[STATE] currentStep=%s", step)
}

func (gc *GeminiClient) GetMode() Mode {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.mode
}

func (gc *GeminiClient) SetMode(mode Mode) {
	gc.mu.Lock()
	gc.mode = mode
	gc.mu.Unlock()
	log.Printf("[STATE] mode=%d", mode)
}

func (gc *GeminiClient) GetPreferredLanguage() string {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.preferredLanguage
}

func (gc *GeminiClient) SetPreferredLanguage(lang string) {
	gc.mu.Lock()
	gc.preferredLanguage = lang
	gc.mu.Unlock()
}

func (gc *GeminiClient) GetUserName() string {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	return gc.userName
}

func (gc *GeminiClient) SetUserName(name string) {
	gc.mu.Lock()
	gc.userName = name
	gc.mu.Unlock()
}

func (gc *GeminiClient) GetARResponseChan() chan ChatAndARResponse {
	return gc.arResponseChan
}

// Audio/Transcript broadcasting

func (gc *GeminiClient) sendAudioToFrontend(msg AudioMessage) {
	resp := ChatAndARResponse{
		Type:           "audio",
		AudioDone:      msg.Done,
		SampleRate:     GeminiOutputSampleRate,
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
	case gc.arResponseChan <- resp:
	default:
	}

	gc.mu.Lock()
	subs := append([]chan ChatAndARResponse(nil), gc.audioSubs...)
	gc.mu.Unlock()

	for _, ch := range subs {
		select {
		case ch <- resp:
		default:
		}
	}
}

func (gc *GeminiClient) broadcastTranscriptDelta(delta string, step Step) {
	if delta == "" {
		return
	}

	gc.transcriptBufferLock.Lock()
	defer gc.transcriptBufferLock.Unlock()

	gc.transcriptBuffer.WriteString(delta)

	if gc.transcriptFlushTimer == nil {
		gc.transcriptFlushTimer = time.AfterFunc(250*time.Millisecond, func() {
			gc.flushTranscriptBuffer(step, false)
		})
	}
}

func (gc *GeminiClient) flushTranscriptBuffer(step Step, done bool) {
	gc.transcriptBufferLock.Lock()
	text := gc.transcriptBuffer.String()
	gc.transcriptBuffer.Reset()
	if gc.transcriptFlushTimer != nil {
		gc.transcriptFlushTimer.Stop()
		gc.transcriptFlushTimer = nil
	}
	gc.transcriptBufferLock.Unlock()

	if text == "" && !done {
		return
	}

	gc.mu.Lock()
	subs := append([]chan TranscriptEvent(nil), gc.transcriptSubs...)
	gc.mu.Unlock()

	ev := TranscriptEvent{Text: text, Step: step, Done: done}
	for _, ch := range subs {
		select {
		case ch <- ev:
		default:
		}
	}
}

func (gc *GeminiClient) broadcastTranscriptDone(step Step) {
	gc.flushTranscriptBuffer(step, true)
}

// SubscribeAudio adds a subscriber for audio events
func (gc *GeminiClient) SubscribeAudio(ch chan ChatAndARResponse) {
	gc.mu.Lock()
	gc.audioSubs = append(gc.audioSubs, ch)
	gc.mu.Unlock()
}

// UnsubscribeAudio removes a subscriber
func (gc *GeminiClient) UnsubscribeAudio(ch chan ChatAndARResponse) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	for i, sub := range gc.audioSubs {
		if sub == ch {
			gc.audioSubs = append(gc.audioSubs[:i], gc.audioSubs[i+1:]...)
			break
		}
	}
}

// SubscribeTranscript adds a subscriber for transcript events
func (gc *GeminiClient) SubscribeTranscript(ch chan TranscriptEvent) {
	gc.mu.Lock()
	gc.transcriptSubs = append(gc.transcriptSubs, ch)
	gc.mu.Unlock()
}

// UnsubscribeTranscript removes a subscriber
func (gc *GeminiClient) UnsubscribeTranscript(ch chan TranscriptEvent) {
	gc.mu.Lock()
	defer gc.mu.Unlock()
	for i, sub := range gc.transcriptSubs {
		if sub == ch {
			gc.transcriptSubs = append(gc.transcriptSubs[:i], gc.transcriptSubs[i+1:]...)
			break
		}
	}
}
