package wecom

import (
	"encoding/json"
	"testing"
)

// These tests pin the wire field names/shapes to the official
// @wecom/aibot-node-sdk's TemplateCard schema (dist/types/api.d.ts), not just
// our own reading of the public docs — this is the schema the official
// OpenClaw WeCom plugin (WecomTeam/wecom-openclaw-plugin) itself imports and
// sends. Field-level regressions here (wrong key, wrong nesting) would mean a
// card silently fails to render or the server rejects it.

func mustMarshal(t *testing.T, v interface{}) map[string]interface{} {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var m map[string]interface{}
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("unmarshal back to map: %v", err)
	}
	return m
}

func TestTemplateCard_ButtonInteraction_Shape(t *testing.T) {
	card := &TemplateCard{
		CardType: "button_interaction",
		ButtonSelection: &CardSelectionItem{
			QuestionKey: "q1",
			OptionList:  []CardSelectionOption{{ID: "opt1", Text: "Option 1"}},
		},
		ButtonList: []*CardButton{
			{Text: "Confirm", Style: 1, Key: "confirm"},
		},
	}
	m := mustMarshal(t, card)

	sel, ok := m["button_selection"].(map[string]interface{})
	if !ok {
		t.Fatalf("button_selection missing or wrong shape: %v", m["button_selection"])
	}
	if sel["question_key"] != "q1" {
		t.Fatalf("button_selection.question_key = %v, want q1", sel["question_key"])
	}
	options, ok := sel["option_list"].([]interface{})
	if !ok || len(options) != 1 {
		t.Fatalf("button_selection.option_list = %v, want 1 entry", sel["option_list"])
	}

	buttons, ok := m["button_list"].([]interface{})
	if !ok || len(buttons) != 1 {
		t.Fatalf("button_list = %v, want 1 entry", m["button_list"])
	}
	btn := buttons[0].(map[string]interface{})
	if btn["key"] != "confirm" || btn["text"] != "Confirm" {
		t.Fatalf("button_list[0] = %v", btn)
	}

	// select_list and checkbox belong to other card types, must not leak in.
	if _, present := m["select_list"]; present {
		t.Fatal("select_list should be omitted for button_interaction")
	}
	if _, present := m["checkbox"]; present {
		t.Fatal("checkbox should be omitted for button_interaction")
	}
}

func TestTemplateCard_VoteInteraction_Shape(t *testing.T) {
	card := &TemplateCard{
		CardType: "vote_interaction",
		Checkbox: &CardCheckbox{
			QuestionKey: "vote1",
			Mode:        1, // multi-select
			OptionList: []CardCheckboxOption{
				{ID: "a", Text: "Option A", IsChecked: true},
				{ID: "b", Text: "Option B"},
			},
		},
		SubmitButton: &CardSubmitButton{Text: "Submit", Key: "submit"},
	}
	m := mustMarshal(t, card)

	cb, ok := m["checkbox"].(map[string]interface{})
	if !ok {
		t.Fatalf("checkbox missing or wrong shape: %v", m["checkbox"])
	}
	if cb["question_key"] != "vote1" {
		t.Fatalf("checkbox.question_key = %v", cb["question_key"])
	}
	if mode, _ := cb["mode"].(float64); mode != 1 {
		t.Fatalf("checkbox.mode = %v, want 1", cb["mode"])
	}
	options, ok := cb["option_list"].([]interface{})
	if !ok || len(options) != 2 {
		t.Fatalf("checkbox.option_list = %v, want 2 entries", cb["option_list"])
	}
	first := options[0].(map[string]interface{})
	if first["is_checked"] != true {
		t.Fatalf("checkbox.option_list[0].is_checked = %v, want true", first["is_checked"])
	}

	if _, present := m["button_list"]; present {
		t.Fatal("button_list should be omitted for vote_interaction")
	}
}

func TestTemplateCard_MultipleInteraction_Shape(t *testing.T) {
	card := &TemplateCard{
		CardType: "multiple_interaction",
		SelectList: []*CardSelectionItem{
			{QuestionKey: "q1", Title: "Pick one", OptionList: []CardSelectionOption{{ID: "x", Text: "X"}}},
			{QuestionKey: "q2", OptionList: []CardSelectionOption{{ID: "y", Text: "Y"}}},
		},
		SubmitButton: &CardSubmitButton{Text: "Submit", Key: "submit"},
	}
	m := mustMarshal(t, card)

	list, ok := m["select_list"].([]interface{})
	if !ok || len(list) != 2 {
		t.Fatalf("select_list = %v, want 2 entries", m["select_list"])
	}
	if _, present := m["button_selection"]; present {
		t.Fatal("button_selection should be omitted for multiple_interaction")
	}
}

func TestTemplateCard_QuoteArea_Shape(t *testing.T) {
	card := &TemplateCard{
		CardType: "text_notice",
		QuoteArea: &CardQuoteArea{
			Type:      1,
			URL:       "https://example.test",
			Title:     "Reference",
			QuoteText: "quoted text",
		},
	}
	m := mustMarshal(t, card)

	qa, ok := m["quote_area"].(map[string]interface{})
	if !ok {
		t.Fatalf("quote_area missing or wrong shape: %v", m["quote_area"])
	}
	// Regression check: the official schema's click-type field is "type",
	// not "type_id", and there is no "userid" field on quote_area at all.
	if _, present := qa["type_id"]; present {
		t.Fatal(`quote_area must use "type", not "type_id"`)
	}
	if _, present := qa["userid"]; present {
		t.Fatal("quote_area has no userid field in the real schema")
	}
	if v, _ := qa["type"].(float64); v != 1 {
		t.Fatalf("quote_area.type = %v, want 1", qa["type"])
	}
	if qa["quote_text"] != "quoted text" {
		t.Fatalf("quote_area.quote_text = %v", qa["quote_text"])
	}
}

func TestTemplateCard_ImageTextArea_UsesImageURL(t *testing.T) {
	card := &TemplateCard{
		CardType:      "news_notice",
		ImageTextArea: &CardImageTextArea{Title: "Title", ImageURL: "https://example.test/img.png"},
	}
	m := mustMarshal(t, card)

	area, ok := m["image_text_area"].(map[string]interface{})
	if !ok {
		t.Fatalf("image_text_area missing or wrong shape: %v", m["image_text_area"])
	}
	if _, present := area["thumb_media_id"]; present {
		t.Fatal(`image_text_area must use "image_url", not "thumb_media_id"`)
	}
	if area["image_url"] != "https://example.test/img.png" {
		t.Fatalf("image_text_area.image_url = %v", area["image_url"])
	}
}

func TestTemplateCard_CardAction_MiniprogramJump(t *testing.T) {
	card := &TemplateCard{
		CardType:   "text_notice",
		CardAction: &CardAction{Type: 2, AppID: "wx-app-id", PagePath: "pages/index"},
	}
	m := mustMarshal(t, card)

	action, ok := m["card_action"].(map[string]interface{})
	if !ok {
		t.Fatalf("card_action missing or wrong shape: %v", m["card_action"])
	}
	if action["appid"] != "wx-app-id" || action["pagepath"] != "pages/index" {
		t.Fatalf("card_action miniprogram fields missing: %v", action)
	}
}
