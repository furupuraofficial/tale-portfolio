package realtime

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Step represents the current conversation step
type Step string

const (
	StepIntro        Step = "STEP_INTRO"
	StepAskName      Step = "STEP_ASK_NAME"
	StepGreetByName  Step = "STEP_GREET_BY_NAME"
	StepStartGuide   Step = "STEP_START_GUIDE"
	StepFreeQuestion Step = "STEP_FREE_QUESTION"
	StepAskRule      Step = "STEP_ASK_RULE"
	StepAskShop      Step = "STEP_ASK_SHOP"
)

// Mode represents whether we're in script mode or free conversation mode
type Mode int

const (
	ModeScript Mode = iota // Fixed script part
	ModeFree               // Free conversation
)

// ARAction represents an AR command to be executed on the frontend
type ARAction struct {
	Type   string            `json:"type"`
	Target string            `json:"target,omitempty"`
	Params map[string]string `json:"params,omitempty"`
}

// ScriptStep holds the fixed script and AR actions for each step
type ScriptStep struct {
	Step      Step       `json:"step"`
	Text      string     `json:"text"`
	ARActions []ARAction `json:"arActions"`
}

// ChatAndARResponse is sent to the frontend with text and AR instructions
type ChatAndARResponse struct {
	Type           string     `json:"type,omitempty"` // "text" (default) or "audio"
	Text           string     `json:"text,omitempty"`
	Step           string     `json:"step,omitempty"`
	ARActions      []ARAction `json:"arActions,omitempty"`
	UserName       string     `json:"userName,omitempty"`
	AudioChunk     string     `json:"audioChunk,omitempty"` // base64-encoded PCM16
	AudioDone      bool       `json:"audioDone,omitempty"`  // true when a stream ends
	SampleRate     int        `json:"sampleRate,omitempty"`
	Channels       int        `json:"channels,omitempty"`
	BytesPerSample int        `json:"bytesPerSample,omitempty"`
}

// ScriptTableByLanguage contains scripts for each language
// Supported: ja, en, zh-Hans, ko, fr, es
var ScriptTableByLanguage = map[string]map[Step]ScriptStep{
	// ========== Japanese ==========
	"ja": {
		StepIntro: {
			Step: StepIntro,
			Text: `やっほー！いらっしゃい！私はこの旅館に住んでいる座敷わらしだよ。

			初めてでも安心して過ごせるように、私がお手伝いするね。`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "wave"}},
			},
		},
		StepAskName: {
			Step: StepAskName,
			Text: `お名前を教えてくれる？`,
			ARActions: []ARAction{
				{Type: "SHOW_SPEECH_BUBBLE", Target: "zashiki"},
			},
		},
		StepGreetByName: {
			Step: StepGreetByName,
			Text: `{{NAME}}さん、よろしくね！`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
		StepStartGuide: {
			Step: StepStartGuide,
			Text: `他のルールや施設の使い方も順番に案内していくね。

            何か質問はある？`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
	},
	// ========== English ==========
	"en": {
		StepIntro: {
			Step: StepIntro,
			Text: `Hey there! Welcome! I'm a Zashiki-warashi who lives in this inn.

I'm here to help make your stay comfortable, even if it's your first time here.`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "wave"}},
			},
		},
		StepAskName: {
			Step: StepAskName,
			Text: `Could you tell me your name?`,
			ARActions: []ARAction{
				{Type: "SHOW_SPEECH_BUBBLE", Target: "zashiki"},
			},
		},
		StepGreetByName: {
			Step: StepGreetByName,
			Text: `Nice to meet you, {{NAME}}! `,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
		StepStartGuide: {
			Step: StepStartGuide,
			Text: `I'll guide you through the other rules and how to use the facilities step by step.

Do you have any questions?`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
	},
	// ========== Chinese (Simplified) ==========
	"zh-Hans": {
		StepIntro: {
			Step: StepIntro,
			Text: `嗨！欢迎光临！我是住在这家旅馆里的座敷童子。

即使是第一次来，我也会帮助你舒适地度过。`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "wave"}},
			},
		},
		StepAskName: {
			Step: StepAskName,
			Text: `可以告诉我你的名字吗？`,
			ARActions: []ARAction{
				{Type: "SHOW_SPEECH_BUBBLE", Target: "zashiki"},
			},
		},
		StepGreetByName: {
			Step: StepGreetByName,
			Text: `很高兴认识你，{{NAME}}！`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
		StepStartGuide: {
			Step: StepStartGuide,
			Text: `我会一步一步地介绍其他规则和设施的使用方法。

有什么问题吗？`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
	},
	// ========== Korean ==========
	"ko": {
		StepIntro: {
			Step: StepIntro,
			Text: `안녕! 어서 와! 나는 이 여관에 사는 자시키와라시야.

처음이어도 편안하게 지낼 수 있도록 내가 도와줄게.`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "wave"}},
			},
		},
		StepAskName: {
			Step: StepAskName,
			Text: `이름을 알려줄 수 있어?`,
			ARActions: []ARAction{
				{Type: "SHOW_SPEECH_BUBBLE", Target: "zashiki"},
			},
		},
		StepGreetByName: {
			Step: StepGreetByName,
			Text: `반가워, {{NAME}}! `,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
		StepStartGuide: {
			Step: StepStartGuide,
			Text: `다른 규칙이나 시설 사용법도 차례대로 안내해 줄게.

질문 있어?`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
	},
	// ========== French ==========
	"fr": {
		StepIntro: {
			Step: StepIntro,
			Text: `Salut ! Bienvenue ! Je suis un Zashiki-warashi qui habite dans cette auberge.

Je suis là pour t'aider à passer un bon séjour, même si c'est ta première fois ici.`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "wave"}},
			},
		},
		StepAskName: {
			Step: StepAskName,
			Text: `Peux-tu me dire ton nom ?`,
			ARActions: []ARAction{
				{Type: "SHOW_SPEECH_BUBBLE", Target: "zashiki"},
			},
		},
		StepGreetByName: {
			Step: StepGreetByName,
			Text: `Enchanté, {{NAME}} ! `,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
		StepStartGuide: {
			Step: StepStartGuide,
			Text: `Je vais te guider à travers les autres règles et l'utilisation des installations étape par étape.

Tu as des questions ?`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
	},
	// ========== Spanish ==========
	"es": {
		StepIntro: {
			Step: StepIntro,
			Text: `¡Hola! ¡Bienvenido! Soy un Zashiki-warashi que vive en esta posada.

Estoy aquí para ayudarte a pasar una estancia cómoda, aunque sea tu primera vez aquí.`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "wave"}},
			},
		},
		StepAskName: {
			Step: StepAskName,
			Text: `¿Puedes decirme tu nombre?`,
			ARActions: []ARAction{
				{Type: "SHOW_SPEECH_BUBBLE", Target: "zashiki"},
			},
		},
		StepGreetByName: {
			Step: StepGreetByName,
			Text: `¡Encantado de conocerte, {{NAME}}! `,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
		StepStartGuide: {
			Step: StepStartGuide,
			Text: `Te guiaré por las otras reglas y cómo usar las instalaciones paso a paso.

¿Tienes alguna pregunta?`,
			ARActions: []ARAction{
				{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "bow"}},
			},
		},
	},
}

// GetScriptForLanguage returns the ScriptStep for a given step and language.
// Falls back to English if the language is not found.
func GetScriptForLanguage(step Step, lang string) ScriptStep {
	if table, ok := ScriptTableByLanguage[lang]; ok {
		if script, ok := table[step]; ok {
			return script
		}
	}
	// Fallback to English
	return ScriptTableByLanguage["en"][step]
}

// ScriptTable is kept for backward compatibility (defaults to English)
var ScriptTable = ScriptTableByLanguage["en"]

// FreeConversationActions contains AR actions for different intents in free mode
var FreeConversationActions = map[Step][]ARAction{
	StepFreeQuestion: {
		{Type: "IDLE", Target: "zashiki"},
		{Type: "SHOW_SPEECH_BUBBLE", Target: "zashiki"},
	},
	StepAskRule: {
		{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "explain"}},
		{Type: "SHOW_INFO_PANEL", Target: "rules_panel"},
		{Type: "LOOK_AT", Target: "signboard"},
	},
	StepAskShop: {
		{Type: "PLAY_ANIMATION", Target: "zashiki", Params: map[string]string{"name": "point_outside"}},
		{Type: "SHOW_MAP", Target: "area_map"},
		{Type: "HIGHLIGHT_DIRECTION", Target: "outside"},
	},
}

// IntentKeywords maps keywords to conversation steps for intent classification
var IntentKeywords = map[Step][]string{
	StepAskRule: {
		"ルール", "規則", "決まり", "してもいい", "禁止", "できる", "していい",
		"お風呂", "温泉", "入浴", "シャワー", "風呂",
		"食事", "朝食", "夕食", "ご飯", "食べる",
		"チェックアウト", "退出", "出る",
		"消灯", "門限", "静か", "夜",
		"wifi", "ワイファイ", "インターネット", "パスワード", "ネット",
		"喫煙", "たばこ", "タバコ", "煙草",
		"ゴミ", "ごみ", "リサイクル", "分別",
		// English fallbacks
		"rule", "bath", "meal", "checkout", "wifi", "smoking", "garbage",
	},
	StepAskShop: {
		"近く", "周辺", "近隣", "そば", "付近",
		"店", "お店", "ショップ", "コンビニ", "スーパー",
		"レストラン", "食べ物", "食事", "ご飯", "飲食",
		"観光", "観光地", "見どころ", "スポット", "名所",
		"おすすめ", "オススメ", "推薦", "どこ", "穴場",
		"バス", "電車", "駅", "交通", "アクセス",
		"ATM", "銀行", "お金", "現金",
		"薬局", "病院", "医者", "クリニック",
		// English fallbacks
		"nearby", "shop", "restaurant", "sightseeing", "recommend", "bus", "atm", "pharmacy",
	},
}

// GetNextStep returns the next step in the script sequence
func GetNextStep(current Step) Step {
	switch current {
	case StepIntro:
		return StepAskName
	case StepAskName:
		return StepGreetByName
	case StepGreetByName:
		return StepStartGuide
	case StepStartGuide:
		return StepFreeQuestion
	default:
		return StepFreeQuestion
	}
}

// IsScriptStep returns true if the step is part of the fixed script
func IsScriptStep(step Step) bool {
	switch step {
	case StepIntro, StepAskName, StepGreetByName, StepStartGuide:
		return true
	default:
		return false
	}
}

// ==== Rule-based Q&A System ====

// Rule represents a rule-based Q&A entry
type Rule struct {
	ID        string     `json:"id"`
	Keywords  []string   `json:"keywords"`
	Question  string     `json:"question"`
	Answer    string     `json:"answer"`
	ARActions []ARAction `json:"arActions"`
}

var (
	ErrRuleNotFound = errors.New("rule not found")
	ErrRuleExists   = errors.New("rule already exists")
	ErrRuleInvalid  = errors.New("invalid rule")
)

var (
	ruleDatabaseMu sync.RWMutex
	ruleDatabase   []Rule
	rulesFilename  string
)

// LoadRules loads rules from rules.json file
func LoadRules(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read rules file: %w", err)
	}

	var rules []Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("failed to parse rules JSON: %w", err)
	}

	ruleDatabaseMu.Lock()
	ruleDatabase = cloneRules(rules)
	rulesFilename = filename
	ruleDatabaseMu.Unlock()

	fmt.Printf("✓ Loaded %d rules from %s\n", len(rules), filename)
	return nil
}

// ListRules returns a copy safe for callers to inspect or encode.
func ListRules() []Rule {
	ruleDatabaseMu.RLock()
	defer ruleDatabaseMu.RUnlock()
	return cloneRules(ruleDatabase)
}

func CreateRule(rule Rule) (Rule, error) {
	ruleDatabaseMu.Lock()
	defer ruleDatabaseMu.Unlock()

	rule = normalizeRule(rule)
	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule-%d", time.Now().UnixNano())
	}
	if err := validateRule(rule); err != nil {
		return Rule{}, err
	}
	if findRuleIndexLocked(rule.ID) >= 0 {
		return Rule{}, fmt.Errorf("%w: %q", ErrRuleExists, rule.ID)
	}

	ruleDatabase = append(ruleDatabase, cloneRule(rule))
	if err := saveRulesLocked(); err != nil {
		ruleDatabase = ruleDatabase[:len(ruleDatabase)-1]
		return Rule{}, err
	}
	return cloneRule(rule), nil
}

func UpdateRule(id string, rule Rule) (Rule, error) {
	ruleDatabaseMu.Lock()
	defer ruleDatabaseMu.Unlock()

	idx := findRuleIndexLocked(id)
	if idx < 0 {
		return Rule{}, fmt.Errorf("%w: %q", ErrRuleNotFound, id)
	}
	rule.ID = id
	rule = normalizeRule(rule)
	if err := validateRule(rule); err != nil {
		return Rule{}, err
	}

	previous := ruleDatabase[idx]
	ruleDatabase[idx] = cloneRule(rule)
	if err := saveRulesLocked(); err != nil {
		ruleDatabase[idx] = previous
		return Rule{}, err
	}
	return cloneRule(rule), nil
}

func DeleteRule(id string) error {
	ruleDatabaseMu.Lock()
	defer ruleDatabaseMu.Unlock()

	idx := findRuleIndexLocked(id)
	if idx < 0 {
		return fmt.Errorf("%w: %q", ErrRuleNotFound, id)
	}
	previous := cloneRules(ruleDatabase)
	ruleDatabase = append(ruleDatabase[:idx], ruleDatabase[idx+1:]...)
	if err := saveRulesLocked(); err != nil {
		ruleDatabase = previous
		return err
	}
	return nil
}

func normalizeRule(rule Rule) Rule {
	rule.ID = strings.TrimSpace(rule.ID)
	rule.Question = strings.TrimSpace(rule.Question)
	rule.Answer = strings.TrimSpace(rule.Answer)
	keywords := make([]string, 0, len(rule.Keywords))
	for _, keyword := range rule.Keywords {
		if keyword = strings.TrimSpace(keyword); keyword != "" {
			keywords = append(keywords, keyword)
		}
	}
	rule.Keywords = keywords
	return rule
}

func validateRule(rule Rule) error {
	switch {
	case rule.ID == "":
		return fmt.Errorf("%w: rule id is required", ErrRuleInvalid)
	case len(rule.Keywords) == 0:
		return fmt.Errorf("%w: at least one keyword is required", ErrRuleInvalid)
	case rule.Question == "":
		return fmt.Errorf("%w: question is required", ErrRuleInvalid)
	case rule.Answer == "":
		return fmt.Errorf("%w: answer is required", ErrRuleInvalid)
	default:
		return nil
	}
}

func findRuleIndexLocked(id string) int {
	for i := range ruleDatabase {
		if ruleDatabase[i].ID == id {
			return i
		}
	}
	return -1
}

func saveRulesLocked() error {
	if rulesFilename == "" {
		return fmt.Errorf("rules database has not been loaded")
	}
	data, err := json.MarshalIndent(ruleDatabase, "", "  ")
	if err != nil {
		return fmt.Errorf("encode rules: %w", err)
	}
	tmp, err := os.CreateTemp(filepath.Dir(rulesFilename), ".rules-*.json")
	if err != nil {
		return fmt.Errorf("create temporary rules file: %w", err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(append(data, '\n')); err != nil {
		tmp.Close()
		return fmt.Errorf("write rules: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close rules file: %w", err)
	}
	if err := os.Rename(tmpName, rulesFilename); err != nil {
		return fmt.Errorf("replace rules file: %w", err)
	}
	return nil
}

func cloneRules(rules []Rule) []Rule {
	cloned := make([]Rule, len(rules))
	for i := range rules {
		cloned[i] = cloneRule(rules[i])
	}
	return cloned
}

func cloneRule(rule Rule) Rule {
	rule.Keywords = append([]string(nil), rule.Keywords...)
	rule.ARActions = append([]ARAction(nil), rule.ARActions...)
	for i := range rule.ARActions {
		if rule.ARActions[i].Params != nil {
			rule.ARActions[i].Params = make(map[string]string, len(rule.ARActions[i].Params))
			for key, value := range rule.ARActions[i].Params {
				rule.ARActions[i].Params[key] = value
			}
		}
	}
	return rule
}

// GenerateRulesPrompt creates a system prompt section from rules.json
// This allows GPT to answer questions about inn rules without keyword matching
func GenerateRulesPrompt() string {
	rules := ListRules()
	if len(rules) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("【Inn Rules & Facilities Information】\n")
	sb.WriteString("Use this information to answer guest questions. Respond naturally in the user's language.\n\n")

	for _, rule := range rules {
		sb.WriteString(fmt.Sprintf("## %s\n", rule.ID))
		sb.WriteString(fmt.Sprintf("%s\n\n", rule.Answer))
	}

	return sb.String()
}

// FindMatchingRule searches for a rule that matches the user's input
func FindMatchingRule(userInput string) *Rule {
	lower := strings.ToLower(userInput)

	// Check each rule's keywords
	rules := ListRules()
	for i := range rules {
		rule := &rules[i]
		for _, keyword := range rule.Keywords {
			if strings.Contains(lower, strings.ToLower(keyword)) {
				return rule
			}
		}
	}

	return nil
}
