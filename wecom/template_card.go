package wecom

import (
	"context"
	"fmt"
)

// TemplateCard represents a WeCom AI Bot template card.
// Field shapes are taken from the official @wecom/aibot-node-sdk's TemplateCard
// interface, not just the public API docs, since some fields (e.g. which
// sub-object belongs to which card_type) are only annotated there.
// Full documentation: https://developer.work.weixin.qq.com/document/path/91770
type TemplateCard struct {
	// CardType is one of: text_notice, news_notice, button_interaction,
	// vote_interaction, multiple_interaction.
	CardType string `json:"card_type"`

	Source          *CardSource          `json:"source,omitempty"`
	ActionMenu      *CardActionMenu      `json:"action_menu,omitempty"`
	MainTitle       *CardMainTitle       `json:"main_title,omitempty"`
	EmphasisContent *CardEmphasisContent `json:"emphasis_content,omitempty"`
	// QuoteArea and EmphasisContent are not meant to be used together.
	QuoteArea         *CardQuoteArea           `json:"quote_area,omitempty"`
	SubTitleText      string                   `json:"sub_title_text,omitempty"`
	HorizontalContent []*CardHorizontalContent `json:"horizontal_content_list,omitempty"` // max 6
	JumpList          []*CardJump              `json:"jump_list,omitempty"`               // max 3
	CardAction        *CardAction              `json:"card_action,omitempty"`

	// news_notice only.
	CardImage       *CardImage             `json:"card_image,omitempty"`
	ImageTextArea   *CardImageTextArea     `json:"image_text_area,omitempty"`
	VerticalContent []*CardVerticalContent `json:"vertical_content_list,omitempty"` // max 4

	// button_interaction only.
	ButtonSelection *CardSelectionItem `json:"button_selection,omitempty"`
	ButtonList      []*CardButton      `json:"button_list,omitempty"` // max 6

	// vote_interaction only.
	Checkbox *CardCheckbox `json:"checkbox,omitempty"`

	// multiple_interaction only (max 3 selectors).
	SelectList []*CardSelectionItem `json:"select_list,omitempty"`

	// vote_interaction / multiple_interaction only.
	SubmitButton *CardSubmitButton `json:"submit_button,omitempty"`

	// TaskID must be unique per bot; only digits/letters/"_-@", max 128 bytes.
	TaskID string `json:"task_id,omitempty"`

	Feedback *Feedback `json:"feedback,omitempty"`
}

// CardSource identifies the card source.
type CardSource struct {
	IconURL   string `json:"icon_url,omitempty"`
	Desc      string `json:"desc,omitempty"`       // suggested <= 13 chars
	DescColor int    `json:"desc_color,omitempty"` // 0=grey(default) 1=black 2=red 3=green
}

// CardActionMenu is the card's top-right "more actions" menu.
type CardActionMenu struct {
	Desc       string                `json:"desc"`
	ActionList []CardActionMenuEntry `json:"action_list"` // 1-3 entries
}

// CardActionMenuEntry is one entry in a CardActionMenu.
type CardActionMenuEntry struct {
	Text string `json:"text"`
	Key  string `json:"key"` // unique, max 1024 bytes
}

// CardMainTitle is the card's main title.
type CardMainTitle struct {
	Title string `json:"title,omitempty"` // suggested <= 26 chars
	Desc  string `json:"desc,omitempty"`  // suggested <= 30 chars
}

// CardEmphasisContent is the "key data" emphasis style.
type CardEmphasisContent struct {
	Title string `json:"title,omitempty"` // suggested <= 10 chars
	Desc  string `json:"desc,omitempty"`  // suggested <= 15 chars
}

// CardQuoteArea is the quoted-reference style area.
type CardQuoteArea struct {
	Type      int    `json:"type,omitempty"` // 0/unset=none 1=url 2=miniprogram
	URL       string `json:"url,omitempty"`  // required when Type==1
	AppID     string `json:"appid,omitempty"`
	PagePath  string `json:"pagepath,omitempty"`
	Title     string `json:"title,omitempty"`
	QuoteText string `json:"quote_text,omitempty"`
}

// CardHorizontalContent is a "sub-title + text" row.
// Max 6 entries per card.
type CardHorizontalContent struct {
	Type    int    `json:"type,omitempty"`   // 0/unset=text 1=url 3=member detail
	KeyName string `json:"keyname"`          // suggested <= 5 chars
	Value   string `json:"value,omitempty"`  // suggested <= 26 chars
	URL     string `json:"url,omitempty"`    // required when Type==1
	UserID  string `json:"userid,omitempty"` // required when Type==3
}

// CardJump is a "jump guide" style entry. Max 3 entries per card.
type CardJump struct {
	Type     int    `json:"type,omitempty"` // 0/unset=none 1=url 2=miniprogram 3=智能问答
	Title    string `json:"title"`          // suggested <= 13 chars
	URL      string `json:"url,omitempty"`  // required when Type==1
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
	Question string `json:"question,omitempty"` // required when Type==3, max 200 bytes
}

// CardAction is the whole-card click action.
type CardAction struct {
	Type     int    `json:"type"` // 0=none 1=url 2=miniprogram
	URL      string `json:"url,omitempty"`
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
}

// CardImage is the header image for news_notice cards.
type CardImage struct {
	URL string `json:"url"`
	// AspectRatio must be in (1.3, 2.25); defaults to 1.3 if unset.
	AspectRatio float64 `json:"aspect_ratio,omitempty"`
}

// CardImageTextArea is the left-image/right-text area for news_notice cards.
type CardImageTextArea struct {
	Type     int    `json:"type,omitempty"` // 0/unset=none 1=url 2=miniprogram
	URL      string `json:"url,omitempty"`
	AppID    string `json:"appid,omitempty"`
	PagePath string `json:"pagepath,omitempty"`
	Title    string `json:"title,omitempty"`
	Desc     string `json:"desc,omitempty"`
	ImageURL string `json:"image_url"`
}

// CardVerticalContent is a secondary vertical content block for news_notice
// cards. Max 4 entries per card.
type CardVerticalContent struct {
	Title string `json:"title"`          // suggested <= 26 chars
	Desc  string `json:"desc,omitempty"` // suggested <= 112 chars
}

// CardSubmitButton is the submit button for vote_interaction /
// multiple_interaction cards.
type CardSubmitButton struct {
	Text string `json:"text"` // suggested <= 10 chars
	Key  string `json:"key"`  // max 1024 bytes
}

// CardSelectionItem is a dropdown selector, used by button_interaction's
// ButtonSelection (single object) and multiple_interaction's SelectList
// (up to 3 objects).
type CardSelectionItem struct {
	QuestionKey string                `json:"question_key"`      // unique, max 1024 bytes
	Title       string                `json:"title,omitempty"`   // suggested <= 13 chars
	Disable     bool                  `json:"disable,omitempty"` // only meaningful on card update
	SelectedID  string                `json:"selected_id,omitempty"`
	OptionList  []CardSelectionOption `json:"option_list"` // 1-10 options
}

// CardSelectionOption is one option within a CardSelectionItem.
type CardSelectionOption struct {
	ID   string `json:"id"`   // unique, max 128 bytes
	Text string `json:"text"` // suggested <= 10 chars
}

// CardButton is a button in button_interaction's ButtonList (max 6).
type CardButton struct {
	Text  string `json:"text"`            // suggested <= 10 chars
	Style int    `json:"style,omitempty"` // 1-4, defaults to 1
	Key   string `json:"key"`             // unique, max 1024 bytes
}

// CardCheckbox is the choice-question style for vote_interaction cards.
type CardCheckbox struct {
	QuestionKey string               `json:"question_key"`      // max 1024 bytes
	Disable     bool                 `json:"disable,omitempty"` // only meaningful on card update
	Mode        int                  `json:"mode,omitempty"`    // 0=single(default) 1=multi
	OptionList  []CardCheckboxOption `json:"option_list"`       // 1-20 options
}

// CardCheckboxOption is one option within a CardCheckbox.
type CardCheckboxOption struct {
	ID        string `json:"id"`                   // unique, max 128 bytes
	Text      string `json:"text"`                 // suggested <= 11 chars
	IsChecked bool   `json:"is_checked,omitempty"` // default-selected
}

// SendTemplateCardReply sends a template card as a reply.
func (b *WecomBot) SendTemplateCardReply(ctx context.Context, reqID string, card *TemplateCard) error {
	if b.client == nil || !b.client.IsConnected() {
		return fmt.Errorf("wecom client not connected")
	}

	body := map[string]interface{}{
		"msgtype":       MsgTypeTemplateCard,
		"template_card": card,
	}
	_, err := b.client.SendReply(ctx, reqID, body)
	return err
}

// SendWelcomeText sends a text welcome message. Must be within 5s of enter_chat.
func (b *WecomBot) SendWelcomeText(ctx context.Context, reqID, text string) error {
	if b.client == nil || !b.client.IsConnected() {
		return fmt.Errorf("wecom client not connected")
	}

	body := map[string]interface{}{
		"msgtype": "text",
		"text":    map[string]string{"content": text},
	}
	_, err := b.client.SendWelcome(ctx, reqID, body)
	return err
}

// SendWelcomeCard sends a template card welcome message. Must be within 5s of enter_chat.
func (b *WecomBot) SendWelcomeCard(ctx context.Context, reqID string, card *TemplateCard) error {
	if b.client == nil || !b.client.IsConnected() {
		return fmt.Errorf("wecom client not connected")
	}

	body := map[string]interface{}{
		"msgtype":       "template_card",
		"template_card": card,
	}
	_, err := b.client.SendWelcome(ctx, reqID, body)
	return err
}

// UpdateTemplateCard updates a template card. Must be within 5s of card event.
func (b *WecomBot) UpdateTemplateCard(ctx context.Context, reqID string, card *TemplateCard, userIDs []string) error {
	if b.client == nil || !b.client.IsConnected() {
		return fmt.Errorf("wecom client not connected")
	}

	body := map[string]interface{}{
		"response_type": "update_template_card",
		"template_card": card,
	}
	if len(userIDs) > 0 {
		body["userids"] = userIDs
	}
	_, err := b.client.SendUpdateCard(ctx, reqID, body)
	return err
}
