# Zashiki-warashi Conversation State Machine

This implementation uses a conversation state machine to manage the flow between scripted introductions and free conversation, with AR action coordination.

## Architecture Overview

### 1. Conversation Modes

- **Script Mode (`ModeScript`)**: Fixed, predefined conversation flow
- **Free Mode (`ModeFree`)**: AI-powered free conversation with intent classification

### 2. Conversation Steps

#### Script Steps (Fixed Flow)
1. `STEP_INTRO`: Initial greeting
2. `STEP_ASK_NAME`: Ask user's name
3. `STEP_GREET_BY_NAME`: Personalized greeting with user's name
4. `STEP_START_GUIDE`: Transition to free conversation

#### Free Conversation Steps (Intent-based)
- `STEP_FREE_QUESTION`: General conversation
- `STEP_ASK_RULE`: Questions about inn rules/facilities
- `STEP_ASK_SHOP`: Questions about nearby shops/attractions

### 3. AR Actions

Each step includes AR commands sent to the frontend:

```json
{
  "type": "PLAY_ANIMATION",
  "target": "zashiki",
  "params": {"name": "wave"}
}
```

**Available AR Action Types:**
- `PLAY_ANIMATION`: Play character animation
- `MOVE_TO_NODE`: Move camera/character to location
- `IDLE`: Return to idle state
- `SHOW_SPEECH_BUBBLE`: Display speech bubble
- `HIGHLIGHT_OBJECT`: Highlight an object
- `SHOW_INFO_PANEL`: Show information panel
- `LOOK_AT`: Look at target
- `SHOW_MAP`: Display area map
- `HIGHLIGHT_DIRECTION`: Point to direction

### 4. Communication Flow

```
Backend (Go) <--WebSocket--> AR Frontend (Swift/RealityKit)
     |
     ├─ OpenAI Realtime API (Voice)
     └─ State Machine (Script/Free)
```

## File Structure

```
tale-backend/
├── cmd/
│   └── voice-chat/
│       └── main.go              # Entry point
├── internal/
│   ├── ar/
│   │   └── server.go            # WebSocket server for AR frontend
│   ├── realtime/
│   │   ├── client.go            # Realtime API + state management
│   │   └── state.go             # State definitions and script table
│   └── api/
│       └── swift.go             # REST API for iOS
├── configs/
│   └── rules.json               # Rules database
└── docs/
    └── STATE_MACHINE_README.md  # This file
```

## Key Components

### State Machine (`internal/realtime/state.go`)

```go
type Step string
type Mode int

// Fixed scripts with AR actions
var ScriptTableByLanguage = map[string]map[Step]ScriptStep{...}

// Intent classification keywords
var IntentKeywords = map[Step][]string{...}
```

### Realtime Client Functions

- `SpeakStep(step Step)`: Speak fixed script for a step
- `AdvanceToNextStep()`: Move to next script step
- `HandleUserInput(transcript)`: Process user input based on mode/step
- `ClassifyIntent(message)`: Determine user intent in free mode
- `StartScriptedConversation()`: Begin the scripted flow

## Usage

### Running the System

```bash
# Make sure .env has OPENAI_API_KEY
go run ./cmd/voice-chat
```

The system will:
1. Start AR WebSocket server on port 8080
2. Connect to OpenAI Realtime API
3. Begin scripted introduction automatically
4. Transition to free conversation after the script

### AR Frontend Connection

Connect to: `ws://localhost:8080/ws`

**Message Format from Backend:**
```json
{
  "text": "Hello there, welcome!",
  "step": "STEP_INTRO",
  "arActions": [
    {"type": "PLAY_ANIMATION", "target": "zashiki", "params": {"name": "wave"}},
    {"type": "MOVE_TO_NODE", "target": "entrance"}
  ],
  "userName": "Alice"
}
```

### Adding New Steps

1. Add step constant in `internal/realtime/state.go`:
```go
const StepNewFeature Step = "STEP_NEW_FEATURE"
```

2. Add to ScriptTableByLanguage:
```go
StepNewFeature: {
    Step: StepNewFeature,
    Text: `Your fixed script here`,
    ARActions: []ARAction{
        {Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "gesture"}},
    },
}
```

3. Update `GetNextStep()` if needed

### Adding New Intent Keywords

In `internal/realtime/state.go`:
```go
var IntentKeywords = map[Step][]string{
    StepAskRule: {"keyword1", "keyword2", ...},
}
```

## Implementation Details

### Script Mode Flow

1. System starts with `StepIntro`
2. User hears greeting, responds
3. System extracts name from `STEP_ASK_NAME`
4. Continues through fixed steps
5. At `STEP_START_GUIDE`, transitions to free mode

### Free Mode Flow

1. User speaks
2. Intent classified by keywords
3. Appropriate AR actions triggered
4. GPT generates natural response
5. Response sent to user + AR frontend

### Name Extraction

Simple pattern matching removes common phrases:
- "My name is [name]"
- "I'm [name]"
- "Call me [name]"

### State Transitions

All transitions logged to console:
```
[State transition: STEP_INTRO -> STEP_ASK_NAME]
[User name extracted: Alice]
[Entering free conversation mode]
```

## Testing

### Test Script Mode
1. Run the system
2. Respond to name question
3. Watch it progress through all steps

### Test Free Mode
1. Wait for "Do you have any questions?"
2. Ask about rules: "What are the bath rules?"
3. Ask about shops: "Where can I eat nearby?"
4. Check AR actions in logs

### Test AR Connection
```bash
# Use wscat to test WebSocket
npm install -g wscat
wscat -c ws://localhost:8080/ws
```

## Future Enhancements

1. **GPT-based Intent Classification**: Use GPT to classify intent instead of keywords
2. **Multi-language Support**: Adapt AR actions based on user language
3. **Dynamic Script Loading**: Load scripts from JSON files
4. **State Persistence**: Save/restore conversation state
5. **Advanced AR Coordination**: Timing control for animations
