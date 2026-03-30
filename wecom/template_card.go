package wecom

import (
	"context"
	"fmt"
)

// TemplateCard represents a WeCom AI Bot template card.
// Only the commonly-used card types and fields are defined.
// Full documentation: https://developer.work.weixin.qq.com/document/path/91770
type TemplateCard struct {
	CardType string `json:"card_type"` // "text_notice", "news_notice", "button_interaction", "vote_interaction", "multiple_interaction"

	// Common fields
	Source            *CardSource              `json:"source,omitempty"`
	MainTitle         *CardMainTitle           `json:"main_title,omitempty"`
	QuoteArea         *CardQuoteArea           `json:"quote_area,omitempty"`
	SubTitleText      string                   `json:"sub_title_text,omitempty"`
	HorizontalContent []*CardHorizontalContent `json:"horizontal_content_list,omitempty"`
	JumpList          []*CardJump              `json:"jump_list,omitempty"`
	CardAction        *CardAction              `json:"card_action,omitempty"`

	// Image text area (for news_notice)
	ImageTextArea *CardImageTextArea `json:"image_text_area,omitempty"`

	// Button selection (for button_interaction)
	ButtonSelection string `json:"button_selection,omitempty"`

	// Submit button (for interaction cards)
	SubmitButton *CardSubmitButton `json:"submit_button,omitempty"`

	// Checkbox (for multiple_interaction)
	Checkbox *CardCheckbox `json:"checkbox,omitempty"`

	// Task ID for card event correlation
	TaskID string `json:"task_id,omitempty"`
}

// CardSource identifies the card source.
type CardSource struct {
	Description string `json:"desc"`
	DescColor   int    `json:"desc_color,omitempty"`
}

// CardMainTitle is the card's main title.
type CardMainTitle struct {
	Title string `json:"title"`
	Desc  string `json:"desc,omitempty"`
}

// CardQuoteArea shows a quoted reference.
type CardQuoteArea struct {
	TypeID int    `json:"type_id,omitempty"` // 0=none, 1=userid, 2=partyid
	UserID string `json:"userid,omitempty"`
}

// CardHorizontalContent is a horizontal content row.
type CardHorizontalContent struct {
	Type    int    `json:"type"` // 1=text, 2=image
	KeyName string `json:"keyname"`
	Value   string `json:"value"`
	URL     string `json:"url,omitempty"`
}

// CardJump is a jump action entry.
type CardJump struct {
	Title string `json:"title"`
	URL   string `json:"url"`
	Type  int    `json:"type,omitempty"` // 0=web, 1=miniprogram
	AppID string `json:"appid,omitempty"`
}

// CardAction is the card-level action.
type CardAction struct {
	Type int    `json:"type"` // 1=open URL, 2=miniprogram
	URL  string `json:"url,omitempty"`
}

// CardImageTextArea is the image+text area for news_notice cards.
type CardImageTextArea struct {
	ThumbMediaID string `json:"thumb_media_id"`
	URL          string `json:"url,omitempty"`
	Title        string `json:"title"`
	Desc         string `json:"desc,omitempty"`
}

// CardSubmitButton is the submit button for interaction cards.
type CardSubmitButton struct {
	Text string `json:"text"`
	Key  string `json:"key"`
}

// CardCheckbox is the checkbox for multiple_interaction cards.
type CardCheckbox struct {
	OptionList  []*CardSelectionItem `json:"option_list"`
	QuestionKey string               `json:"question_key"`
}

// CardSelectionItem is a selectable option.
type CardSelectionItem struct {
	ID   string `json:"id"`
	Text string `json:"text"`
}

// SendTemplateCardReply sends a template card as a reply.
func (a *ActionsAdapter) SendTemplateCardReply(ctx context.Context, reqID string, card *TemplateCard) error {
	client := a.gateway.GetClient("")
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("wecom client not connected")
	}

	body := map[string]interface{}{
		"msgtype":       MsgTypeTemplateCard,
		"template_card": card,
	}
	return client.SendReply(ctx, reqID, body)
}

// SendWelcomeText sends a text welcome message. Must be within 5s of enter_chat.
func (a *ActionsAdapter) SendWelcomeText(ctx context.Context, reqID, text string) error {
	client := a.gateway.GetClient("")
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("wecom client not connected")
	}

	body := map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": text},
	}
	return client.SendWelcome(ctx, reqID, body)
}

// SendWelcomeCard sends a template card welcome message. Must be within 5s of enter_chat.
func (a *ActionsAdapter) SendWelcomeCard(ctx context.Context, reqID string, card *TemplateCard) error {
	client := a.gateway.GetClient("")
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("wecom client not connected")
	}

	body := map[string]interface{}{
		"msgtype":       "template_card",
		"template_card": card,
	}
	return client.SendWelcome(ctx, reqID, body)
}

// UpdateTemplateCard updates a template card. Must be within 5s of card event.
func (a *ActionsAdapter) UpdateTemplateCard(ctx context.Context, reqID string, card *TemplateCard, userIDs []string) error {
	client := a.gateway.GetClient("")
	if client == nil || !client.IsConnected() {
		return fmt.Errorf("wecom client not connected")
	}

	body := map[string]interface{}{
		"response_type": "update_template_card",
		"template_card": card,
	}
	if len(userIDs) > 0 {
		body["userids"] = userIDs
	}
	return client.SendUpdateCard(ctx, reqID, body)
}
