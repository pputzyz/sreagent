package lark

import (
	"encoding/json"
	"fmt"
)

// Card V2 schema and builder — implements Lark Card JSON 2.0.
// See https://open.larksuite.com/document/uAjLw4CM/ukzMukzMukzM/feishu-cards/card-json-v2-structure

const (
	maxCardComponents = 200
	maxCardBytes      = 30 * 1024 // 30 KB
)

// CardV2 is the top-level Card 2.0 message body.
type CardV2 struct {
	Schema   string       `json:"schema"`              // must be "2.0"
	Config   *CardV2Config `json:"config,omitempty"`
	Header   *CardV2Header `json:"header,omitempty"`
	Body     *CardV2Body   `json:"body,omitempty"`
	CardLink *CardLink     `json:"card_link,omitempty"`
}

type CardV2Config struct {
	WideScreenMode bool   `json:"wide_screen_mode,omitempty"`
	EnableForward  bool   `json:"enable_forward,omitempty"`
	WidthMode      string `json:"width_mode,omitempty"` // "default" | "fill"
	Summary        *CardV2Text `json:"summary,omitempty"` // chat list preview
}

type CardLink struct {
	URL         string `json:"url"`
	AndroidURL  string `json:"android_url,omitempty"`
	IOSURL      string `json:"ios_url,omitempty"`
	PCURL       string `json:"pc_url,omitempty"`
}

type CardV2Header struct {
	Title    *CardV2Text `json:"title"`
	Subtitle *CardV2Text `json:"subtitle,omitempty"`
	Template string      `json:"template,omitempty"` // red/orange/yellow/green/blue/indigo/purple/carmine/violet/wathet/turquoise/grey
	UdIcon   *UdIcon     `json:"ud_icon,omitempty"`
}

type UdIcon struct {
	Tag  string `json:"tag"`            // "standard-icon" | "custom-icon"
	Token string `json:"token,omitempty"` // standard icon token
}

type CardV2Text struct {
	Tag     string `json:"tag"`               // "plain_text" | "lark_md"
	Content string `json:"content"`
}

type CardV2Body struct {
	Elements []interface{} `json:"elements"`
}

// --- Element types ---

// MarkdownElement renders Lark Markdown in a card.
type MarkdownElement struct {
	Tag     string `json:"tag"` // "markdown"
	Content string `json:"content"`
}

// DividerElement is a horizontal rule.
type DividerElement struct {
	Tag string `json:"tag"` // "hr"
}

// ColumnSetElement is a multi-column layout container.
type ColumnSetElement struct {
	Tag     string           `json:"tag"` // "column_set"`
	FlexMode string          `json:"flex_mode,omitempty"` // "none" | "stretch" | "bisect"
	Columns  []ColumnElement `json:"columns"`
}

type ColumnElement struct {
	Tag      string        `json:"tag"` // "column"
	Width    string        `json:"width,omitempty"` // "weighted" | "auto"
	Weight   int           `json:"weight,omitempty"` // 1-10, used when width="weighted"
	Elements []interface{} `json:"elements"`
}

// NoteElement is a footer note area.
type NoteElement struct {
	Tag      string        `json:"tag"` // "note"
	Elements []interface{} `json:"elements"`
}

// ImageElement displays an image.
type ImageElement struct {
	Tag      string `json:"tag"` // "img"
	ImgKey   string `json:"img_key"`
	Alt      *CardV2Text `json:"alt"`
	Title    *CardV2Text `json:"title,omitempty"`
}

// ActionElement wraps interactive components (buttons, selects, forms).
type ActionElement struct {
	Tag      string        `json:"tag"` // "action"
	Actions  []interface{} `json:"actions"`
}

// FormElement wraps form inputs with a submit button.
type FormElement struct {
	Tag      string        `json:"tag"` // "form"
	Name     string        `json:"name"`
	Elements []interface{} `json:"elements"`
}

// CollapsiblePanelElement is a collapsible panel container.
type CollapsiblePanelElement struct {
	Tag            string        `json:"tag"` // "collapsible_panel"`
	Expanded       bool          `json:"expanded"`
	Header         *PanelHeader  `json:"header"`
	Border         *PanelBorder  `json:"border,omitempty"`
	Elements       []interface{} `json:"elements"`
}

type PanelHeader struct {
	Title    *CardV2Text `json:"title"`
	Subtitle *CardV2Text `json:"subtitle,omitempty"`
	Icon     *UdIcon     `json:"icon,omitempty"`
}

type PanelBorder struct {
	Color string `json:"color,omitempty"`
}

// ChartElement embeds a VChart spec for inline charts.
type ChartElement struct {
	Tag          string      `json:"tag"` // "chart"
	AspectRatio  string      `json:"aspect_ratio,omitempty"` // "1:1" | "2:1" | "4:3" | "16:9"
	ChartSpec    interface{} `json:"chart_spec"` // VChart JSON spec
}

// --- Button components ---

// ButtonV2 is a Card 2.0 button with behaviors array.
type ButtonV2 struct {
	Tag      string      `json:"tag"` // "button"
	Text     *CardV2Text `json:"text"`
	Type     string      `json:"type,omitempty"` // "primary" | "danger" | "default"
	Size     string      `json:"size,omitempty"`  // "tiny" | "small" | "medium" | "large"
	Behaviors []Behavior `json:"behaviors,omitempty"`
	Name     string      `json:"name,omitempty"` // form field name
	Disabled bool        `json:"disabled,omitempty"`
}

// Behavior defines a button action.
type Behavior struct {
	Type  string      `json:"type"` // "callback" | "open_url"
	Value interface{} `json:"value,omitempty"` // callback payload
	DefaultURL string `json:"default_url,omitempty"` // open_url target
}

// SelectMenuV2 is a Card 2.0 select dropdown.
type SelectMenuV2 struct {
	Tag         string        `json:"tag"` // "select_static" | "select_person"
	Placeholder *CardV2Text   `json:"placeholder"`
	Name        string        `json:"name"`
	Options     []OptionV2    `json:"options,omitempty"`
	Value       interface{}   `json:"value,omitempty"`
}

type OptionV2 struct {
	Text  *CardV2Text `json:"text"`
	Value string      `json:"value"`
}

// InputV2 is a Card 2.0 text input.
type InputV2 struct {
	Tag         string     `json:"tag"` // "input"`
	Placeholder *CardV2Text `json:"placeholder"`
	Name        string     `json:"name"`
	MaxLines    int        `json:"max_lines,omitempty"`
}

// PersonElement displays a person avatar + name.
type PersonElement struct {
	Tag     string   `json:"tag"` // "person"`
	Mode    string   `json:"mode,omitempty"` // "name" | "avatar"
	Size    string   `json:"size,omitempty"` // "medium" | "large"
	OpenIDs []string `json:"open_ids"`
}

// --- Builder ---

// CardV2Builder provides a fluent API for building Card 2.0 messages.
type CardV2Builder struct {
	card     CardV2
	elements []interface{}
	size     int // estimated JSON size
}

// NewCardV2Builder creates a new Card 2.0 builder.
func NewCardV2Builder() *CardV2Builder {
	return &CardV2Builder{
		card: CardV2{Schema: "2.0"},
	}
}

// Config sets the card config.
func (b *CardV2Builder) Config(cfg *CardV2Config) *CardV2Builder {
	b.card.Config = cfg
	return b
}

// Header sets the card header with a color template.
func (b *CardV2Builder) Header(title, template string) *CardV2Builder {
	b.card.Header = &CardV2Header{
		Title:    &CardV2Text{Tag: "plain_text", Content: title},
		Template: template,
	}
	return b
}

// HeaderWithSubtitle sets the card header with title, subtitle, and color.
func (b *CardV2Builder) HeaderWithSubtitle(title, subtitle, template string) *CardV2Builder {
	b.card.Header = &CardV2Header{
		Title:    &CardV2Text{Tag: "plain_text", Content: title},
		Subtitle: &CardV2Text{Tag: "plain_text", Content: subtitle},
		Template: template,
	}
	return b
}

// CardLink sets the card link (whole card is clickable).
func (b *CardV2Builder) CardLink(url string) *CardV2Builder {
	b.card.CardLink = &CardLink{URL: url}
	return b
}

// AddMarkdown appends a markdown element.
func (b *CardV2Builder) AddMarkdown(content string) *CardV2Builder {
	b.elements = append(b.elements, MarkdownElement{Tag: "markdown", Content: content})
	b.size += len(content) + 30
	return b
}

// AddDivider appends a horizontal divider.
func (b *CardV2Builder) AddDivider() *CardV2Builder {
	b.elements = append(b.elements, DividerElement{Tag: "hr"})
	b.size += 20
	return b
}

// AddChart appends a chart element with a VChart spec.
func (b *CardV2Builder) AddChart(aspectRatio string, chartSpec interface{}) *CardV2Builder {
	b.elements = append(b.elements, ChartElement{
		Tag:         "chart",
		AspectRatio: aspectRatio,
		ChartSpec:   chartSpec,
	})
	b.size += 200 // rough estimate for chart
	return b
}

// AddCollapsiblePanel appends a collapsible panel.
func (b *CardV2Builder) AddCollapsiblePanel(title string, expanded bool, innerElements ...interface{}) *CardV2Builder {
	b.elements = append(b.elements, CollapsiblePanelElement{
		Tag:      "collapsible_panel",
		Expanded: expanded,
		Header:   &PanelHeader{Title: &CardV2Text{Tag: "plain_text", Content: title}},
		Elements: innerElements,
	})
	b.size += len(title) + 50
	return b
}

// AddActions appends an action row with buttons.
func (b *CardV2Builder) AddActions(buttons ...interface{}) *CardV2Builder {
	b.elements = append(b.elements, ActionElement{Tag: "action", Actions: buttons})
	b.size += 50
	return b
}

// AddPerson appends a person display element.
func (b *CardV2Builder) AddPerson(openIDs ...string) *CardV2Builder {
	b.elements = append(b.elements, PersonElement{Tag: "person", Mode: "name", OpenIDs: openIDs})
	b.size += 50
	return b
}

// AddNote appends a footer note.
func (b *CardV2Builder) AddNote(elements ...interface{}) *CardV2Builder {
	b.elements = append(b.elements, NoteElement{Tag: "note", Elements: elements})
	b.size += 30
	return b
}

// Build finalizes the card and returns it. Returns an error if the estimated size exceeds 30KB.
func (b *CardV2Builder) Build() (*CardV2, error) {
	b.card.Body = &CardV2Body{Elements: b.elements}

	if len(b.elements) > maxCardComponents {
		return nil, fmt.Errorf("card has %d elements, limit is %d", len(b.elements), maxCardComponents)
	}

	data, err := json.Marshal(b.card)
	if err != nil {
		return nil, fmt.Errorf("marshal card: %w", err)
	}
	if len(data) > maxCardBytes {
		return nil, fmt.Errorf("card size %d bytes exceeds %d byte limit", len(data), maxCardBytes)
	}

	return &b.card, nil
}

// BuildJSON returns the card as a JSON string (for CardKit CreateCardEntity).
func (b *CardV2Builder) BuildJSON() (string, error) {
	card, err := b.Build()
	if err != nil {
		return "", err
	}
	data, err := json.Marshal(card)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// --- Helper constructors ---

// NewMarkdown creates a markdown element.
func NewMarkdown(content string) MarkdownElement {
	return MarkdownElement{Tag: "markdown", Content: content}
}

// NewButton creates a callback button.
func NewButton(text, action, style string, value map[string]interface{}) ButtonV2 {
	btn := ButtonV2{
		Tag:  "button",
		Text: &CardV2Text{Tag: "plain_text", Content: text},
		Type: style,
		Size: "small",
	}
	if action == "callback" {
		btn.Behaviors = []Behavior{{Type: "callback", Value: value}}
	} else if action == "open_url" {
		if url, ok := value["default_url"].(string); ok {
			btn.Behaviors = []Behavior{{Type: "open_url", DefaultURL: url}}
		}
	}
	return btn
}

// SeverityToTemplate maps alert severity to Card 2.0 header color template.
func SeverityToTemplate(severity string) string {
	switch severity {
	case "critical":
		return "red"
	case "error":
		return "orange"
	case "warning":
		return "yellow"
	case "info":
		return "blue"
	default:
		return "grey"
	}
}

// StatusToTemplate maps alert event status to Card 2.0 header color template.
func StatusToTemplate(status string) string {
	switch status {
	case "firing":
		return "red"
	case "acknowledged":
		return "orange"
	case "assigned":
		return "blue"
	case "silenced":
		return "yellow"
	case "resolved":
		return "green"
	case "closed":
		return "grey"
	default:
		return "grey"
	}
}
