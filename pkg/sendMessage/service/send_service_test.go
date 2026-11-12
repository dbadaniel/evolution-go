package send_service

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"image"
	"image/png"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"

	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"google.golang.org/protobuf/proto"
)

func TestBuildCTAButtonMessageUsesLeanNativeFlowPayload(t *testing.T) {
	buttonParams := []map[string]string{
		{"display_text": "Site", "url": "https://example.com", "merchant_url": "https://example.com"},
		{"display_text": "Ligar", "phone_number": "+5511999999999"},
		{"display_text": "Copiar", "id": "coupon", "copy_code": "PROMO10"},
	}
	buttonNames := []string{"cta_url", "cta_call", "cta_copy"}
	buttons := make([]*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton, len(buttonNames))
	for i := range buttonNames {
		params, err := json.Marshal(buttonParams[i])
		if err != nil {
			t.Fatalf("marshal button params: %v", err)
		}
		paramsJSON := string(params)
		buttons[i] = &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
			Name:             proto.String(buttonNames[i]),
			ButtonParamsJSON: &paramsJSON,
		}
	}

	data := &ButtonStruct{
		Title:       "Oferta",
		Description: "Escolha uma acao",
		Footer:      "Rodape",
	}
	messageParamsJSON := `{"from":"api","templateId":123}`
	secret := bytes.Repeat([]byte{0x2a}, 32)

	msg := buildCTAButtonMessage(data, buttons, messageParamsJSON, secret)
	if msg.GetDocumentWithCaptionMessage() == nil || msg.GetDocumentWithCaptionMessage().GetMessage() == nil {
		t.Fatal("CTA must be wrapped in DocumentWithCaptionMessage")
	}
	interactive := msg.GetDocumentWithCaptionMessage().GetMessage().GetInteractiveMessage()
	if interactive == nil {
		t.Fatal("CTA wrapper must contain InteractiveMessage")
	}
	if interactive.GetBody().GetText() != "*Oferta*\n\nEscolha uma acao\n" {
		t.Fatalf("unexpected body: %q", interactive.GetBody().GetText())
	}
	if interactive.GetFooter().GetText() != data.Footer {
		t.Fatalf("unexpected footer: %q", interactive.GetFooter().GetText())
	}
	if interactive.GetHeader() != nil {
		t.Fatal("CTA payload must not contain InteractiveMessage.Header")
	}
	if interactive.GetContextInfo() != nil {
		t.Fatal("CTA payload must not contain InteractiveMessage.ContextInfo")
	}
	nativeFlow := interactive.GetNativeFlowMessage()
	if nativeFlow == nil || nativeFlow.GetMessageVersion() != 1 {
		t.Fatal("CTA payload must contain NativeFlowMessage version 1")
	}
	if nativeFlow.GetMessageParamsJSON() != messageParamsJSON {
		t.Fatalf("unexpected message params: %q", nativeFlow.GetMessageParamsJSON())
	}
	if len(nativeFlow.GetButtons()) != len(buttonNames) {
		t.Fatalf("unexpected button count: %d", len(nativeFlow.GetButtons()))
	}
	for i, button := range nativeFlow.GetButtons() {
		if button.GetName() != buttonNames[i] {
			t.Fatalf("button %d name = %q, want %q", i, button.GetName(), buttonNames[i])
		}
		var gotParams map[string]string
		if err := json.Unmarshal([]byte(button.GetButtonParamsJSON()), &gotParams); err != nil {
			t.Fatalf("button %d params are invalid JSON: %v", i, err)
		}
		if len(gotParams) != len(buttonParams[i]) {
			t.Fatalf("button %d params = %#v, want %#v", i, gotParams, buttonParams[i])
		}
		for key, want := range buttonParams[i] {
			if gotParams[key] != want {
				t.Fatalf("button %d param %q = %q, want %q", i, key, gotParams[key], want)
			}
		}
	}
	if !bytes.Equal(msg.GetMessageContextInfo().GetMessageSecret(), secret) {
		t.Fatal("CTA payload must preserve MessageSecret")
	}
}

func TestBuildCTAButtonMessageOmitsEmptyFooterAndExtraFields(t *testing.T) {
	msg := buildCTAButtonMessage(&ButtonStruct{Title: "Oferta"}, nil, `{}`, []byte("secret"))
	interactive := msg.GetDocumentWithCaptionMessage().GetMessage().GetInteractiveMessage()
	if interactive.GetFooter() != nil {
		t.Fatal("empty footer must be omitted for client compatibility")
	}
	if interactive.GetHeader() != nil || interactive.GetContextInfo() != nil {
		t.Fatal("lean CTA payload must omit header and internal context info")
	}
}

func TestBuildPixNativeFlowButtonUsesWebCompatiblePaymentInfoPayload(t *testing.T) {
	button, err := buildPixNativeFlowButton(Button{
		Currency: "BRL",
		Name:     "Loja de Teste",
		KeyType:  "CPF",
		Key:      "12345678909",
	})
	if err != nil {
		t.Fatalf("build PIX button: %v", err)
	}
	if button.GetName() != "payment_info" {
		t.Fatalf("button name = %q, want payment_info", button.GetName())
	}

	var params map[string]interface{}
	if err := json.Unmarshal([]byte(button.GetButtonParamsJSON()), &params); err != nil {
		t.Fatalf("unmarshal PIX button params: %v", err)
	}
	for _, key := range []string{"order", "total_amount", "reference_id", "payment_settings", "external_payment_configurations", "additional_note", "currency", "type"} {
		if _, ok := params[key]; !ok {
			t.Errorf("PIX params missing %q: %#v", key, params)
		}
	}
	settings, ok := params["payment_settings"].([]interface{})
	if !ok || len(settings) != 2 {
		t.Fatalf("payment_settings = %#v, want static PIX and disabled cards", params["payment_settings"])
	}
	setting, ok := settings[0].(map[string]interface{})
	if !ok || setting["type"] != "pix_static_code" {
		t.Fatalf("payment setting = %#v, want pix_static_code", settings[0])
	}
	pix, ok := setting["pix_static_code"].(map[string]interface{})
	if !ok || pix["merchant_name"] != "Loja de Teste" || pix["key"] != "12345678909" || pix["key_type"] != "CPF" {
		t.Fatalf("pix_static_code = %#v, want supplied static key details", setting["pix_static_code"])
	}
	order, ok := params["order"].(map[string]interface{})
	if !ok || order["order_type"] != "ORDER_WITHOUT_AMOUNT" || order["status"] != "payment_requested" {
		t.Fatalf("order = %#v, want a payment request without amount", params["order"])
	}
	if params["currency"] != "BRL" || params["type"] != "physical-goods" {
		t.Fatalf("currency/type = %#v/%#v, want BRL/physical-goods", params["currency"], params["type"])
	}
}

func TestNormalizePixKeyValidatesAndNormalizesCPF(t *testing.T) {
	got, err := NormalizePixKey("cpf", "123.456.789-09")
	if err != nil {
		t.Fatalf("NormalizePixKey(valid CPF) error = %v", err)
	}
	if got != "12345678909" {
		t.Fatalf("NormalizePixKey(valid CPF) = %q, want digits only", got)
	}

	for _, key := range []string{"12345678900", "11111111111", "12345"} {
		if _, err := NormalizePixKey("CPF", key); err == nil {
			t.Errorf("NormalizePixKey(%q) succeeded, want invalid CPF error", key)
		}
	}
}

func TestBuildPixButtonMessageUsesTopLevelPaymentInfoFlow(t *testing.T) {
	button, err := buildPixNativeFlowButton(Button{Name: "Loja", KeyType: "random", Key: "test-key"})
	if err != nil {
		t.Fatalf("build PIX button: %v", err)
	}
	secret := bytes.Repeat([]byte{0x3c}, 32)
	msg := buildPixButtonMessage(&ButtonStruct{
		Title:       "Pix de teste",
		Description: "Use a chave abaixo",
		Footer:      "Loja",
	}, button, secret)

	if msg.GetInteractiveMessage() == nil || msg.GetDocumentWithCaptionMessage() != nil || msg.GetViewOnceMessage() != nil {
		t.Fatal("PIX must use a top-level InteractiveMessage without legacy or ViewOnce wrappers")
	}
	interactive := msg.GetInteractiveMessage()
	if interactive.GetBody().GetText() != "*Pix de teste*\n\nUse a chave abaixo" || interactive.GetFooter().GetText() != "Loja" {
		t.Fatalf("unexpected PIX body/footer: %q / %q", interactive.GetBody().GetText(), interactive.GetFooter().GetText())
	}
	nativeFlow := interactive.GetNativeFlowMessage()
	if nativeFlow == nil || nativeFlow.GetMessageVersion() != 1 || nativeFlow.GetMessageParamsJSON() != "{}" {
		t.Fatalf("PIX native flow = %#v, want version 1 and empty params", nativeFlow)
	}
	if len(nativeFlow.GetButtons()) != 1 || nativeFlow.GetButtons()[0].GetName() != "payment_info" {
		t.Fatalf("PIX buttons = %#v, want one payment_info button", nativeFlow.GetButtons())
	}
	contextInfo := msg.GetMessageContextInfo()
	if !bytes.Equal(contextInfo.GetMessageSecret(), secret) {
		t.Fatal("PIX payload must preserve MessageSecret")
	}
	if contextInfo.GetDeviceListMetadata() == nil || contextInfo.GetDeviceListMetadataVersion() != 2 {
		t.Fatal("PIX payload must include version 2 device-list metadata for linked clients")
	}
}

func TestBuildReplyButtonMessageUsesTopLevelNativeFlow(t *testing.T) {
	data := &ButtonStruct{
		Title:       "Confirmacao",
		Description: "Escolha uma opcao",
		Footer:      "Responda abaixo",
		Buttons: []Button{
			{Type: "reply", Id: `yes-"quoted"`, DisplayText: `Sim "agora"`},
			{Type: "reply", Id: `no\\later`, DisplayText: `Nao \\ depois`},
		},
	}
	buttons, err := buildQuickReplyButtons(data.Buttons)
	if err != nil {
		t.Fatalf("build quick replies: %v", err)
	}

	secret := bytes.Repeat([]byte{0x7a}, 32)
	messageParamsJSON := `{"from":"api","templateId":123}`
	msg := buildReplyButtonMessage(data, buttons, messageParamsJSON, secret)

	if msg.GetViewOnceMessage() != nil || msg.GetViewOnceMessageV2() != nil {
		t.Fatal("reply buttons must not use a ViewOnce envelope")
	}
	if msg.GetInteractiveMessage() == nil || msg.GetButtonsMessage() != nil || msg.GetDocumentWithCaptionMessage() != nil {
		t.Fatal("modern reply buttons must not use ButtonsMessage or DocumentWithCaptionMessage")
	}
	innerMessage := msg
	if innerMessage.GetButtonsMessage() != nil || innerMessage.GetDocumentWithCaptionMessage() != nil {
		t.Fatal("reply message must not contain a legacy reply envelope")
	}
	interactive := innerMessage.GetInteractiveMessage()
	if interactive == nil {
		t.Fatal("reply message must contain top-level InteractiveMessage")
	}
	if interactive.GetBody().GetText() != "*Confirmacao*\n\nEscolha uma opcao" {
		t.Fatalf("unexpected reply body: %q", interactive.GetBody().GetText())
	}
	if interactive.GetFooter().GetText() != data.Footer {
		t.Fatalf("unexpected reply footer: %q", interactive.GetFooter().GetText())
	}

	nativeFlow := interactive.GetNativeFlowMessage()
	if nativeFlow == nil || nativeFlow.GetMessageVersion() != 1 {
		t.Fatal("reply payload must contain NativeFlowMessage version 1")
	}
	if nativeFlow.GetMessageParamsJSON() != messageParamsJSON {
		t.Fatalf("unexpected message params: %q", nativeFlow.GetMessageParamsJSON())
	}
	if len(nativeFlow.GetButtons()) != len(data.Buttons) {
		t.Fatalf("unexpected button count: %d", len(nativeFlow.GetButtons()))
	}
	for i, button := range nativeFlow.GetButtons() {
		if button.GetName() != "quick_reply" {
			t.Fatalf("button %d name = %q, want quick_reply", i, button.GetName())
		}
		var params map[string]string
		if err := json.Unmarshal([]byte(button.GetButtonParamsJSON()), &params); err != nil {
			t.Fatalf("button %d params are invalid JSON: %v", i, err)
		}
		if params["id"] != data.Buttons[i].Id || params["display_text"] != data.Buttons[i].DisplayText {
			t.Fatalf("button %d params = %#v, want id/display text preserved", i, params)
		}
	}
	messageContextInfo := innerMessage.GetMessageContextInfo()
	if !bytes.Equal(messageContextInfo.GetMessageSecret(), secret) {
		t.Fatal("reply payload must preserve MessageSecret")
	}
	if messageContextInfo.GetDeviceListMetadata() == nil {
		t.Fatal("reply payload must include device list metadata")
	}
	if messageContextInfo.GetDeviceListMetadataVersion() != 2 {
		t.Fatalf("device list metadata version = %d, want 2", messageContextInfo.GetDeviceListMetadataVersion())
	}

	quoted := QuotedStruct{MessageID: "quoted-message-id", Participant: "5511999999999@s.whatsapp.net"}
	applyInteractiveQuotedMessage(msg, quoted)
	contextInfo := interactive.GetContextInfo()
	if contextInfo.GetStanzaID() != quoted.MessageID || contextInfo.GetParticipant() != quoted.Participant || contextInfo.GetQuotedMessage() == nil {
		t.Fatalf("quoted context not preserved: %#v", contextInfo)
	}
	if !bytes.Equal(innerMessage.GetMessageContextInfo().GetMessageSecret(), secret) {
		t.Fatal("applying quoted context must not replace MessageSecret")
	}

	mentionedJIDs := []string{"5511888888888@s.whatsapp.net", "5511777777777@s.whatsapp.net"}
	applyInteractiveMentions(msg, mentionedJIDs)
	if got := interactive.GetContextInfo().GetMentionedJID(); len(got) != len(mentionedJIDs) || got[0] != mentionedJIDs[0] || got[1] != mentionedJIDs[1] {
		t.Fatalf("mentions not preserved on inner interactive message: %#v", got)
	}
	if interactive.GetContextInfo().GetStanzaID() != quoted.MessageID || interactive.GetContextInfo().GetParticipant() != quoted.Participant || interactive.GetContextInfo().GetQuotedMessage() == nil {
		t.Fatal("applying mentions must preserve quoted context")
	}
	if !bytes.Equal(innerMessage.GetMessageContextInfo().GetMessageSecret(), secret) || innerMessage.GetMessageContextInfo().GetDeviceListMetadataVersion() != 2 {
		t.Fatal("applying mentions must not replace inner message metadata")
	}
}

func TestUnwrapViewOnceMessageSupportsNestedVersions(t *testing.T) {
	inner := &waE2E.Message{InteractiveMessage: &waE2E.InteractiveMessage{}}
	wrapped := &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{Message: &waE2E.Message{
			ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: inner},
		}},
	}

	if got := unwrapViewOnceMessage(wrapped); got != inner {
		t.Fatalf("unwrapViewOnceMessage() = %p, want %p", got, inner)
	}
}

func TestUnwrapViewOnceMessageRejectsCycles(t *testing.T) {
	first := &waE2E.Message{}
	second := &waE2E.Message{}
	first.ViewOnceMessage = &waE2E.FutureProofMessage{Message: second}
	second.ViewOnceMessageV2 = &waE2E.FutureProofMessage{Message: first}

	if got := unwrapViewOnceMessage(first); got != nil {
		t.Fatalf("unwrapViewOnceMessage() = %p, want nil for cycle", got)
	}
}

func TestUnwrapViewOnceMessageEnforcesDepthBound(t *testing.T) {
	inner := &waE2E.Message{InteractiveMessage: &waE2E.InteractiveMessage{}}
	withinBound := inner
	for range maxViewOnceMessageDepth {
		withinBound = &waE2E.Message{ViewOnceMessage: &waE2E.FutureProofMessage{Message: withinBound}}
	}
	if got := unwrapViewOnceMessage(withinBound); got != inner {
		t.Fatalf("unwrapViewOnceMessage() = %p, want %p at depth bound", got, inner)
	}

	beyondBound := &waE2E.Message{ViewOnceMessageV2: &waE2E.FutureProofMessage{Message: withinBound}}
	if got := unwrapViewOnceMessage(beyondBound); got != nil {
		t.Fatalf("unwrapViewOnceMessage() = %p, want nil beyond depth bound", got)
	}
}

func TestBuildReplyButtonAdditionalNodesUsesOfficialClientShape(t *testing.T) {
	timestamp := time.Unix(1_799_000_123, 0)
	decisionID := []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13}
	nodes := *buildReplyButtonAdditionalNodes(timestamp, decisionID)

	if len(nodes) != 1 || nodes[0].Tag != "biz" {
		t.Fatalf("nodes = %#v, want one biz node", nodes)
	}
	biz := nodes[0]
	if biz.Attrs["actual_actors"] != "2" || biz.Attrs["host_storage"] != "2" || biz.Attrs["privacy_mode_ts"] != "1799000123" {
		t.Fatalf("unexpected biz attrs: %#v", biz.Attrs)
	}
	children, ok := biz.Content.([]waBinary.Node)
	if !ok || len(children) != 2 {
		t.Fatalf("biz content = %#v, want interactive and quality_control", biz.Content)
	}
	interactive := children[0]
	if interactive.Tag != "interactive" || interactive.Attrs["type"] != "native_flow" || interactive.Attrs["v"] != "1" {
		t.Fatalf("unexpected interactive node: %#v", interactive)
	}
	nativeFlow, ok := interactive.Content.([]waBinary.Node)
	if !ok || len(nativeFlow) != 1 || nativeFlow[0].Tag != "native_flow" || nativeFlow[0].Attrs["v"] != "9" || nativeFlow[0].Attrs["name"] != "mixed" {
		t.Fatalf("unexpected native flow node: %#v", interactive.Content)
	}
	qualityControl := children[1]
	if qualityControl.Tag != "quality_control" || qualityControl.Attrs["source_type"] != "third_party" {
		t.Fatalf("unexpected quality control node: %#v", qualityControl)
	}
	encodedDecisionID, ok := qualityControl.Attrs["decision_id"].(string)
	if !ok || len(encodedDecisionID) != 40 {
		t.Fatalf("decision_id = %#v, want 40 hexadecimal characters", qualityControl.Attrs["decision_id"])
	}
	decodedDecisionID, err := hex.DecodeString(encodedDecisionID)
	if err != nil || !bytes.Equal(decodedDecisionID, decisionID) {
		t.Fatalf("decision_id = %q, decode error = %v", encodedDecisionID, err)
	}
	decisionSource, ok := qualityControl.Content.([]waBinary.Node)
	if !ok || len(decisionSource) != 1 || decisionSource[0].Tag != "decision_source" || decisionSource[0].Attrs["value"] != "df" {
		t.Fatalf("unexpected decision source: %#v", qualityControl.Content)
	}
	for _, node := range nodes {
		if node.Tag == "bot" {
			t.Fatal("modern reply transport must not include a bot node")
		}
	}
}

func TestNewReplyButtonAdditionalNodesGeneratesRuntimeMetadata(t *testing.T) {
	before := time.Now().Unix()
	nodesPointer, err := newReplyButtonAdditionalNodes()
	after := time.Now().Unix()
	if err != nil {
		t.Fatalf("build runtime reply nodes: %v", err)
	}
	if nodesPointer == nil || len(*nodesPointer) != 1 || (*nodesPointer)[0].Tag != "biz" {
		t.Fatalf("nodes = %#v, want one biz node", nodesPointer)
	}

	biz := (*nodesPointer)[0]
	timestampText, ok := biz.Attrs["privacy_mode_ts"].(string)
	if !ok {
		t.Fatalf("privacy_mode_ts = %#v, want string", biz.Attrs["privacy_mode_ts"])
	}
	timestamp, err := strconv.ParseInt(timestampText, 10, 64)
	if err != nil || timestamp < before || timestamp > after {
		t.Fatalf("privacy_mode_ts = %q, want Unix timestamp between %d and %d (parse error: %v)", timestampText, before, after, err)
	}

	children, ok := biz.Content.([]waBinary.Node)
	if !ok {
		t.Fatalf("biz content = %#v, want child nodes", biz.Content)
	}
	var qualityControl *waBinary.Node
	for i := range children {
		if children[i].Tag == "quality_control" {
			qualityControl = &children[i]
			break
		}
	}
	if qualityControl == nil {
		t.Fatal("biz node must contain quality_control")
	}
	decisionID, ok := qualityControl.Attrs["decision_id"].(string)
	if !ok || len(decisionID) != 40 {
		t.Fatalf("decision_id = %#v, want 40 hexadecimal characters", qualityControl.Attrs["decision_id"])
	}
	decoded, err := hex.DecodeString(decisionID)
	if err != nil || len(decoded) != 20 {
		t.Fatalf("decision_id = %q, want 20 decodable bytes (decode error: %v)", decisionID, err)
	}
}

func TestUsesModernReplyButtonsOnlyForTextReplies(t *testing.T) {
	tests := []struct {
		name          string
		data          ButtonStruct
		hasReply      bool
		hasOtherTypes bool
		hasPix        bool
		want          bool
	}{
		{name: "text reply", hasReply: true, want: true},
		{name: "reply with image", data: ButtonStruct{ImageUrl: "https://example.com/image.jpg"}, hasReply: true},
		{name: "reply with video", data: ButtonStruct{VideoUrl: "https://example.com/video.mp4"}, hasReply: true},
		{name: "reply with image and video", data: ButtonStruct{ImageUrl: "https://example.com/image.jpg", VideoUrl: "https://example.com/video.mp4"}, hasReply: true},
		{name: "reply with whitespace image URL", data: ButtonStruct{ImageUrl: " \t "}, hasReply: true},
		{name: "reply with whitespace video URL", data: ButtonStruct{VideoUrl: " \n "}, hasReply: true},
		{name: "reply mixed with CTA", hasReply: true, hasOtherTypes: true},
		{name: "reply mixed with Pix", hasReply: true, hasPix: true},
		{name: "CTA", hasOtherTypes: true},
		{name: "Pix", hasPix: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := usesModernReplyButtons(&tt.data, tt.hasReply, tt.hasOtherTypes, tt.hasPix); got != tt.want {
				t.Fatalf("usesModernReplyButtons() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildButtonAdditionalNodes(t *testing.T) {
	tests := []struct {
		name           string
		hasPix         bool
		hasCTA         bool
		number         string
		wantNativeFlow string
		wantBot        bool
	}{
		{
			name:           "CTA individual omits bot",
			hasCTA:         true,
			number:         "5511999999999@s.whatsapp.net",
			wantNativeFlow: "mixed",
		},
		{
			name:           "CTA group omits bot",
			hasCTA:         true,
			number:         "120363000000000000@g.us",
			wantNativeFlow: "mixed",
		},
		{
			name:   "Pix individual uses payment biz node",
			hasPix: true,
			number: "5511999999999@s.whatsapp.net",
		},
		{
			name:           "non-CTA individual preserves bot",
			number:         "5511999999999@s.whatsapp.net",
			wantNativeFlow: "mixed",
			wantBot:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nodes := *buildButtonAdditionalNodes(tt.hasPix, tt.hasCTA, tt.number)
			if tt.hasPix {
				if len(nodes) != 1 || nodes[0].Tag != "biz" || nodes[0].Attrs["native_flow_name"] != "payment_info" || nodes[0].Content != nil {
					t.Fatalf("PIX nodes = %#v, want a flat payment_info biz node", nodes)
				}
				return
			}
			if len(nodes) == 0 || nodes[0].Tag != "biz" {
				t.Fatalf("first node = %#v, want biz", nodes)
			}

			interactiveNodes, ok := nodes[0].Content.([]waBinary.Node)
			if !ok || len(interactiveNodes) != 1 || interactiveNodes[0].Tag != "interactive" {
				t.Fatalf("biz content = %#v, want one interactive node", nodes[0].Content)
			}
			if interactiveNodes[0].Attrs["type"] != "native_flow" || interactiveNodes[0].Attrs["v"] != "1" {
				t.Fatalf("interactive attrs = %#v", interactiveNodes[0].Attrs)
			}

			nativeFlowNodes, ok := interactiveNodes[0].Content.([]waBinary.Node)
			if !ok || len(nativeFlowNodes) != 1 || nativeFlowNodes[0].Tag != "native_flow" {
				t.Fatalf("interactive content = %#v, want one native_flow node", interactiveNodes[0].Content)
			}
			if nativeFlowNodes[0].Attrs["name"] != tt.wantNativeFlow {
				t.Fatalf("native flow name = %v, want %q", nativeFlowNodes[0].Attrs["name"], tt.wantNativeFlow)
			}

			gotBot := false
			for _, node := range nodes {
				if node.Tag == "bot" {
					gotBot = true
					if node.Attrs["biz_bot"] != "1" {
						t.Fatalf("bot attrs = %#v, want biz_bot=1", node.Attrs)
					}
				}
			}
			if gotBot != tt.wantBot {
				t.Fatalf("bot node present = %v, want %v; nodes = %#v", gotBot, tt.wantBot, nodes)
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return fn(req)
}

func httpResponse(status int, body, contentType string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
		Header:     http.Header{"Content-Type": []string{contentType}},
	}
}

func TestResolveLinkTextAndMatchedURL(t *testing.T) {
	tests := []struct {
		name        string
		text        string
		explicitURL string
		wantText    string
		wantMatched string
	}{
		{
			name:        "explicit URL already in text",
			text:        "Assista https://www.youtube.com/watch?v=abc agora",
			explicitURL: "https://www.youtube.com/watch?v=abc",
			wantText:    "Assista https://www.youtube.com/watch?v=abc agora",
			wantMatched: "https://www.youtube.com/watch?v=abc",
		},
		{
			name:        "explicit URL is appended exactly once",
			text:        "Assista agora",
			explicitURL: "https://youtu.be/abc",
			wantText:    "Assista agora\nhttps://youtu.be/abc",
			wantMatched: "https://youtu.be/abc",
		},
		{
			name:        "substring is not treated as the exact URL",
			text:        "Assista https://youtu.be/abcdef",
			explicitURL: "https://youtu.be/abc",
			wantText:    "Assista https://youtu.be/abcdef\nhttps://youtu.be/abc",
			wantMatched: "https://youtu.be/abc",
		},
		{
			name:        "uppercase scheme is accepted",
			text:        "Assista HTTP://youtube.com/watch?v=abc",
			explicitURL: "HTTP://youtube.com/watch?v=abc",
			wantText:    "Assista HTTP://youtube.com/watch?v=abc",
			wantMatched: "HTTP://youtube.com/watch?v=abc",
		},
		{
			name:        "uppercase scheme is discovered in text",
			text:        "Assista HTTPS://youtube.com/watch?v=abc",
			wantText:    "Assista HTTPS://youtube.com/watch?v=abc",
			wantMatched: "HTTPS://youtube.com/watch?v=abc",
		},
		{
			name:        "legitimate URL ending in punctuation is preserved",
			text:        "Veja https://example.com/wiki/Foo_(bar)",
			explicitURL: "https://example.com/wiki/Foo_(bar)",
			wantText:    "Veja https://example.com/wiki/Foo_(bar)",
			wantMatched: "https://example.com/wiki/Foo_(bar)",
		},
		{
			name:        "explicit URL legitimately ending in period is not duplicated",
			text:        "Veja https://example.com/path.",
			explicitURL: "https://example.com/path.",
			wantText:    "Veja https://example.com/path.",
			wantMatched: "https://example.com/path.",
		},
		{
			name:        "URL enclosed in parentheses is not duplicated",
			text:        "Veja (https://example.com)",
			explicitURL: "https://example.com",
			wantText:    "Veja (https://example.com)",
			wantMatched: "https://example.com",
		},
		{
			name:        "URL in markdown is not duplicated",
			text:        "[site](https://example.com)",
			explicitURL: "https://example.com",
			wantText:    "[site](https://example.com)",
			wantMatched: "https://example.com",
		},
		{
			name:        "URL in markdown followed by punctuation is not duplicated",
			text:        "[site](https://example.com).",
			explicitURL: "https://example.com",
			wantText:    "[site](https://example.com).",
			wantMatched: "https://example.com",
		},
		{
			name:        "URL after punctuation is not duplicated",
			text:        "link=https://example.com",
			explicitURL: "https://example.com",
			wantText:    "link=https://example.com",
			wantMatched: "https://example.com",
		},
		{
			name:        "longer query is not mistaken for explicit URL",
			text:        "Veja https://example.com/path?a=1&b=2",
			explicitURL: "https://example.com/path?a=1",
			wantText:    "Veja https://example.com/path?a=1&b=2\nhttps://example.com/path?a=1",
			wantMatched: "https://example.com/path?a=1",
		},
		{
			name:        "URL adjacent to emoji is not duplicated",
			text:        "Veja👉https://example.com✅",
			explicitURL: "https://example.com",
			wantText:    "Veja👉https://example.com✅",
			wantMatched: "https://example.com",
		},
		{
			name:        "whitespace-only text becomes only the URL",
			text:        " \n\t ",
			explicitURL: "https://youtu.be/abc",
			wantText:    "https://youtu.be/abc",
			wantMatched: "https://youtu.be/abc",
		},
		{
			name:        "invalid explicit URL falls back to URL in text",
			text:        "Leia https://example.com/noticia",
			explicitURL: "javascript:alert(1)",
			wantText:    "Leia https://example.com/noticia",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "multiline explicit URL is ignored",
			text:        "Texto seguro",
			explicitURL: "https://example.com\r\nconteudo injetado",
			wantText:    "Texto seguro",
			wantMatched: "",
		},
		{
			name:        "explicit URL with internal whitespace is ignored",
			text:        "Texto seguro",
			explicitURL: "https://example.com/parte\tindevida",
			wantText:    "Texto seguro",
			wantMatched: "",
		},
		{
			name:        "explicit URL with internal control is ignored",
			text:        "Texto seguro",
			explicitURL: "https://example.com/parte\x00indevida",
			wantText:    "Texto seguro",
			wantMatched: "",
		},
		{
			name:        "explicit URL with credentials is ignored",
			text:        "Texto seguro",
			explicitURL: "https://usuario:senha@example.com/privado",
			wantText:    "Texto seguro",
			wantMatched: "",
		},
		{
			name:        "consumer line breaks are preserved",
			text:        "Linha um\nLinha dois\n",
			explicitURL: "https://example.com",
			wantText:    "Linha um\nLinha dois\n\nhttps://example.com",
			wantMatched: "https://example.com",
		},
		{
			name:        "URL is discovered in text",
			text:        "Leia https://example.com/noticia hoje",
			wantText:    "Leia https://example.com/noticia hoje",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "URL before unicode whitespace is discovered",
			text:        "Leia https://example.com/noticia\u00a0agora",
			wantText:    "Leia https://example.com/noticia\u00a0agora",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "URL before curly quote is discovered",
			text:        "Leia “https://example.com/noticia” agora",
			wantText:    "Leia “https://example.com/noticia” agora",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "URL before guillemet is discovered",
			text:        "Leia «https://example.com/noticia» agora",
			wantText:    "Leia «https://example.com/noticia» agora",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "URL before unicode ellipsis is discovered",
			text:        "Leia https://example.com/noticia… depois",
			wantText:    "Leia https://example.com/noticia… depois",
			wantMatched: "https://example.com/noticia",
		},
		{
			name:        "URL with apostrophe in path is preserved",
			text:        "Leia https://example.com/o'brien agora",
			wantText:    "Leia https://example.com/o'brien agora",
			wantMatched: "https://example.com/o'brien",
		},
		{
			name:        "URL before zero width separator is discovered",
			text:        "Leia https://example.com/noticia\u200bagora",
			wantText:    "Leia https://example.com/noticia\u200bagora",
			wantMatched: "https://example.com/noticia",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotText, gotMatched := resolveLinkTextAndMatchedURL(tt.text, tt.explicitURL)
			if gotText != tt.wantText || gotMatched != tt.wantMatched {
				t.Fatalf("resolve link = (%q, %q), want (%q, %q)", gotText, gotMatched, tt.wantText, tt.wantMatched)
			}
			if gotMatched != "" && !strings.Contains(gotText, gotMatched) {
				t.Fatalf("resolved text %q does not contain matched URL %q", gotText, gotMatched)
			}
			secondText, secondMatched := resolveLinkTextAndMatchedURL(gotText, tt.explicitURL)
			if secondText != gotText || secondMatched != gotMatched {
				t.Fatalf("resolution must be idempotent, got (%q, %q) after (%q, %q)", secondText, secondMatched, gotText, gotMatched)
			}
		})
	}
}

func TestIsYouTubeURLOnlyAcceptsTrustedHosts(t *testing.T) {
	tests := map[string]bool{
		"https://youtube.com/watch?v=abc":              true,
		"https://www.youtube.com/live/abc":             true,
		"https://music.youtube.com/watch?v=abc":        true,
		"https://youtu.be/abc":                         true,
		"HTTP://YOUTUBE.COM/watch?v=abc":               true,
		"https://youtube.com.evil.example/watch?v=abc": false,
		"https://notyoutube.com/watch?v=abc":           false,
		"https://youtu.be.evil.example/abc":            false,
		"ftp://youtube.com/watch?v=abc":                false,
		"javascript://youtube.com/watch?v=abc":         false,
		"not a URL":                                    false,
	}

	for rawURL, want := range tests {
		if got := isYouTubeURL(rawURL); got != want {
			t.Errorf("isYouTubeURL(%q) = %v, want %v", rawURL, got, want)
		}
	}
}

func TestFetchLinkMetadataUsesYouTubeOEmbed(t *testing.T) {
	targetURL := "https://www.youtube.com/live/live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Host != "www.youtube.com" || req.URL.Path != "/oembed" {
			t.Fatalf("unexpected request URL: %s", req.URL)
		}
		if req.URL.Query().Get("url") != targetURL || req.URL.Query().Get("format") != "json" {
			t.Fatalf("unexpected oEmbed query: %s", req.URL.RawQuery)
		}
		return httpResponse(http.StatusOK, `{"title":"Live correta","author_name":"Canal correto","thumbnail_url":"https://i.ytimg.com/vi/live-id/hqdefault.jpg"}`, "application/json"), nil
	})}

	title, description, imgURL, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch YouTube metadata: %v", err)
	}
	if title != "Live correta" || description != "Canal correto" || imgURL != "https://i.ytimg.com/vi/live-id/hqdefault.jpg" {
		t.Fatalf("unexpected metadata: title=%q description=%q image=%q", title, description, imgURL)
	}
	if requests != 1 {
		t.Fatalf("successful oEmbed must avoid HTML fallback, got %d requests", requests)
	}
}

func TestFetchLinkMetadataFallsBackToHTMLWhenYouTubeOEmbedIsIncomplete(t *testing.T) {
	targetURL := "https://www.youtube.com/live/live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusOK, `{"title":"Live sem thumbnail","author_name":"Canal"}`, "application/json"), nil
		}
		if req.URL.String() != targetURL {
			t.Fatalf("unexpected fallback URL: %s", req.URL)
		}
		return httpResponse(http.StatusOK, `<html><head><meta property="og:title" content="Live via HTML"><meta property="og:description" content="Descricao HTML"><meta property="og:image" content="/thumb.jpg"></head></html>`, "text/html"), nil
	})}

	title, description, imgURL, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch fallback metadata: %v", err)
	}
	if title != "Live via HTML" || description != "Descricao HTML" || imgURL != "https://www.youtube.com/thumb.jpg" {
		t.Fatalf("unexpected fallback metadata: title=%q description=%q image=%q", title, description, imgURL)
	}
	if requests != 2 {
		t.Fatalf("incomplete oEmbed must use HTML fallback, got %d requests", requests)
	}
}

func TestFetchLinkMetadataFallsBackToHTMLWhenYouTubeOEmbedFails(t *testing.T) {
	targetURL := "https://youtu.be/live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusNotFound, `not found`, "text/plain"), nil
		}
		if req.URL.String() != targetURL {
			t.Fatalf("unexpected fallback URL: %s", req.URL)
		}
		return httpResponse(http.StatusOK, `<html><head><meta property="og:title" content="Fallback"><meta property="og:description" content="Pagina publica"><meta property="og:image" content="https://img.example/thumb.jpg"></head></html>`, "text/html"), nil
	})}

	title, description, imgURL, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch fallback metadata after oEmbed failure: %v", err)
	}
	if title != "Fallback" || description != "Pagina publica" || imgURL != "https://img.example/thumb.jpg" {
		t.Fatalf("unexpected fallback metadata: title=%q description=%q image=%q", title, description, imgURL)
	}
	if requests != 2 {
		t.Fatalf("failed oEmbed must use HTML fallback, got %d requests", requests)
	}
}

func TestFetchLinkMetadataFallsBackWhenYouTubeOEmbedJSONIsMalformed(t *testing.T) {
	targetURL := "https://www.youtube.com/watch?v=live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusOK, `{not-json`, "application/json"), nil
		}
		return httpResponse(http.StatusOK, `<html><head><title>Fallback seguro</title></head></html>`, "text/html"), nil
	})}

	title, _, _, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch fallback after malformed JSON: %v", err)
	}
	if title != "Fallback seguro" || requests != 2 {
		t.Fatalf("malformed JSON did not use fallback: title=%q requests=%d", title, requests)
	}
}

func TestFetchLinkMetadataFallsBackWhenYouTubeThumbnailURLIsInvalid(t *testing.T) {
	targetURL := "https://www.youtube.com/watch?v=live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusOK, `{"title":"Live","author_name":"Canal","thumbnail_url":"javascript:alert(1)"}`, "application/json"), nil
		}
		return httpResponse(http.StatusOK, `<html><head><title>Fallback thumbnail segura</title></head></html>`, "text/html"), nil
	})}

	title, _, _, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch fallback after invalid thumbnail URL: %v", err)
	}
	if title != "Fallback thumbnail segura" || requests != 2 {
		t.Fatalf("invalid oEmbed thumbnail did not use fallback: title=%q requests=%d", title, requests)
	}
}

func TestFetchLinkMetadataReservesTimeForHTMLFallback(t *testing.T) {
	targetURL := "https://www.youtube.com/watch?v=live-id"
	var oEmbedRemaining, fallbackRemaining time.Duration
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		deadline, ok := req.Context().Deadline()
		if !ok {
			t.Fatal("metadata request must have a deadline")
		}
		if req.URL.Path == "/oembed" {
			oEmbedRemaining = time.Until(deadline)
			return httpResponse(http.StatusNotFound, "missing", "text/plain"), nil
		}
		fallbackRemaining = time.Until(deadline)
		return httpResponse(http.StatusOK, `<html><head><title>Fallback com tempo</title></head></html>`, "text/html"), nil
	})}

	if _, _, _, err := fetchLinkMetadataWithClient(targetURL, client); err != nil {
		t.Fatalf("fetch metadata with reserved fallback time: %v", err)
	}
	if oEmbedRemaining <= 0 || oEmbedRemaining > 3100*time.Millisecond {
		t.Fatalf("unexpected oEmbed deadline: %v", oEmbedRemaining)
	}
	if fallbackRemaining < 9*time.Second || fallbackRemaining <= oEmbedRemaining {
		t.Fatalf("fallback did not retain the total budget: oEmbed=%v fallback=%v", oEmbedRemaining, fallbackRemaining)
	}
}

func TestFetchLinkMetadataLimitsOnlyYouTubeOEmbedResponse(t *testing.T) {
	targetURL := "https://www.youtube.com/watch?v=live-id"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusOK, strings.Repeat("x", maxLinkMetadataResponseBytes+1), "application/json"), nil
		}
		largePadding := strings.Repeat(" ", maxLinkMetadataResponseBytes+1)
		return httpResponse(http.StatusOK, `<html><head><title>HTML grande aceito</title></head><body>`+largePadding+`</body></html>`, "text/html"), nil
	})}

	title, _, _, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch HTML fallback after oversized oEmbed: %v", err)
	}
	if title != "HTML grande aceito" || requests != 2 {
		t.Fatalf("oversized oEmbed did not use unrestricted HTML fallback: title=%q requests=%d", title, requests)
	}
}

func TestFetchLinkMetadataPreservesBothFailureCauses(t *testing.T) {
	targetURL := "https://youtu.be/live-id"
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/oembed" {
			return httpResponse(http.StatusNotFound, "missing", "text/plain"), nil
		}
		return httpResponse(http.StatusBadGateway, "unavailable", "text/plain"), nil
	})}

	_, _, _, err := fetchLinkMetadataWithClient(targetURL, client)
	if err == nil || !strings.Contains(err.Error(), "oEmbed") || !strings.Contains(err.Error(), "404") || !strings.Contains(err.Error(), "502") {
		t.Fatalf("combined error must preserve both causes, got %v", err)
	}
}

func TestFetchLinkMetadataCommonSiteDoesNotCallYouTubeOEmbed(t *testing.T) {
	targetURL := "https://example.com/article"
	requests := 0
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		requests++
		if req.URL.String() != targetURL {
			t.Fatalf("common site unexpectedly called another endpoint: %s", req.URL)
		}
		return httpResponse(http.StatusOK, `<html><head><title>Example</title><meta name="description" content="Article"><meta property="og:image" content="https://cdn.example.com/thumb.jpg"></head></html>`, "text/html"), nil
	})}

	title, description, imgURL, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch common metadata: %v", err)
	}
	if title != "Example" || description != "Article" || imgURL != "https://cdn.example.com/thumb.jpg" {
		t.Fatalf("unexpected common metadata: title=%q description=%q image=%q", title, description, imgURL)
	}
	if requests != 1 {
		t.Fatalf("common site must make one HTML request, got %d", requests)
	}
}

func TestFetchHTMLLinkMetadataResolvesImageAgainstFinalRedirectURL(t *testing.T) {
	targetURL := "https://short.example/article"
	client := &http.Client{Transport: roundTripFunc(func(req *http.Request) (*http.Response, error) {
		resp := httpResponse(http.StatusOK, `<html><head><title>Redirected</title><meta property="og:image" content="../thumb.jpg"></head></html>`, "text/html")
		finalRequest, err := http.NewRequest(http.MethodGet, "https://final.example/posts/article", nil)
		if err != nil {
			t.Fatalf("build final request: %v", err)
		}
		resp.Request = finalRequest
		return resp, nil
	})}

	_, _, imgURL, err := fetchLinkMetadataWithClient(targetURL, client)
	if err != nil {
		t.Fatalf("fetch redirected metadata: %v", err)
	}
	if imgURL != "https://final.example/thumb.jpg" {
		t.Fatalf("relative image resolved against wrong base: %q", imgURL)
	}
}

func TestNormalizeLinkPreviewThumbnailLimitsDimensionsAndProducesJPEG(t *testing.T) {
	source := image.NewRGBA(image.Rect(0, 0, 1000, 1))
	var input bytes.Buffer
	if err := png.Encode(&input, source); err != nil {
		t.Fatalf("encode source PNG: %v", err)
	}

	thumbnail, width, height, sourceFormat, err := normalizeLinkPreviewThumbnail(input.Bytes(), 200)
	if err != nil {
		t.Fatalf("normalize thumbnail: %v", err)
	}
	if sourceFormat != "png" {
		t.Fatalf("unexpected source format: %q", sourceFormat)
	}
	if width != 200 || height != 1 {
		t.Fatalf("unexpected normalized dimensions: %dx%d", width, height)
	}

	decoded, format, err := image.Decode(bytes.NewReader(thumbnail))
	if err != nil {
		t.Fatalf("decode normalized thumbnail: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("normalized thumbnail must be JPEG, got %q", format)
	}
	if decoded.Bounds().Dx() != int(width) || decoded.Bounds().Dy() != int(height) {
		t.Fatalf("encoded dimensions differ from metadata: %dx%d", decoded.Bounds().Dx(), decoded.Bounds().Dy())
	}
}

func TestNormalizeLinkPreviewThumbnailInvalidImageFallsBack(t *testing.T) {
	thumbnail, width, height, _, err := normalizeLinkPreviewThumbnail([]byte("not-an-image"), 200)
	if err == nil {
		t.Fatal("expected invalid image error")
	}
	if thumbnail != nil || width != 0 || height != 0 {
		t.Fatalf("invalid image must return empty fallback, got bytes=%d dimensions=%dx%d", len(thumbnail), width, height)
	}
}

func TestBuildLinkExtendedTextMessageWithThumbnail(t *testing.T) {
	data := &LinkStruct{
		Text:        "Confira https://example.com/article",
		Title:       "Example",
		Description: "An example article",
	}
	matchedText := "https://example.com/article"
	thumbnail := []byte{0xff, 0xd8, 0xff, 0xd9}

	message := buildLinkExtendedTextMessage(data, matchedText, thumbnail, 200, 120)

	if message.GetText() != data.Text {
		t.Fatalf("unexpected text: %q", message.GetText())
	}
	if message.GetMatchedText() != matchedText {
		t.Fatalf("unexpected matched URL: %q", message.GetMatchedText())
	}
	if message.GetTitle() != data.Title || message.GetDescription() != data.Description {
		t.Fatalf("metadata was not preserved: title=%q description=%q", message.GetTitle(), message.GetDescription())
	}
	if message.GetPreviewType() != waE2E.ExtendedTextMessage_IMAGE {
		t.Fatalf("unexpected preview type: %s", message.GetPreviewType())
	}
	if !bytes.Equal(message.GetJPEGThumbnail(), thumbnail) {
		t.Fatal("thumbnail was not preserved in JPEGThumbnail")
	}
	if message.GetThumbnailWidth() != 200 || message.GetThumbnailHeight() != 120 {
		t.Fatalf("unexpected thumbnail dimensions: %dx%d", message.GetThumbnailWidth(), message.GetThumbnailHeight())
	}
	if message.GetContextInfo() != nil {
		t.Fatal("common link preview must not include context info or external ad reply")
	}

	encoded, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("marshal message: %v", err)
	}
	if bytes.Count(encoded, thumbnail) != 1 {
		t.Fatalf("thumbnail must occur exactly once in protobuf, got %d", bytes.Count(encoded, thumbnail))
	}
}

func TestResolvedMergedLinkPayloadPreservesConsumerMetadataAndSimpleShape(t *testing.T) {
	original := LinkStruct{
		Text:  "Assista a live",
		Url:   "https://youtu.be/live-id",
		Title: "Titulo fornecido",
	}
	resolvedText, matchedText := resolveLinkTextAndMatchedURL(original.Text, original.Url)
	working := original
	working.Text = resolvedText
	working = mergeLinkMetadata(working, "Titulo remoto", "Canal remoto", "https://img.example/thumb.jpg")
	thumbnail := []byte{0xff, 0xd8, 0xff, 0xd9}
	message := buildLinkExtendedTextMessage(&working, matchedText, thumbnail, 200, 120)

	if original.Text != "Assista a live" || original.Description != "" || original.ImgUrl != "" {
		t.Fatalf("pure resolution/merge flow mutated caller input: %+v", original)
	}
	if message.GetTitle() != "Titulo fornecido" || message.GetDescription() != "Canal remoto" {
		t.Fatalf("consumer metadata precedence was not preserved: title=%q description=%q", message.GetTitle(), message.GetDescription())
	}
	if !strings.Contains(message.GetText(), message.GetMatchedText()) {
		t.Fatalf("payload text %q does not contain MatchedText %q", message.GetText(), message.GetMatchedText())
	}
	if message.GetContextInfo() != nil {
		t.Fatal("simple link payload must not contain ContextInfo")
	}
	encoded, err := proto.Marshal(message)
	if err != nil {
		t.Fatalf("marshal integrated payload: %v", err)
	}
	if bytes.Count(encoded, thumbnail) != 1 {
		t.Fatalf("thumbnail must occur exactly once, got %d", bytes.Count(encoded, thumbnail))
	}
}

func TestMergeLinkMetadataPreservesAllConsumerFields(t *testing.T) {
	original := LinkStruct{
		Title:       "Titulo fornecido",
		Description: "Descricao fornecida",
		ImgUrl:      "https://consumer.example/thumb.jpg",
	}
	merged := mergeLinkMetadata(original, "Titulo remoto", "Descricao remota", "https://remote.example/thumb.jpg")

	if merged.Title != original.Title || merged.Description != original.Description || merged.ImgUrl != original.ImgUrl {
		t.Fatalf("consumer metadata was overwritten: %+v", merged)
	}
}

func TestBuildLinkExtendedTextMessageWithoutThumbnail(t *testing.T) {
	data := &LinkStruct{
		Text:        "Confira https://example.com/article",
		Title:       "Example",
		Description: "An example article",
	}
	matchedText := "https://example.com/article"

	message := buildLinkExtendedTextMessage(data, matchedText, nil, 0, 0)

	if message.GetMatchedText() != matchedText {
		t.Fatalf("unexpected matched URL: %q", message.GetMatchedText())
	}
	if message.GetPreviewType() != waE2E.ExtendedTextMessage_PLACEHOLDER {
		t.Fatalf("unexpected preview type: %s", message.GetPreviewType())
	}
	if len(message.GetJPEGThumbnail()) != 0 {
		t.Fatal("message without an image must not include a thumbnail")
	}
	if message.ThumbnailWidth != nil || message.ThumbnailHeight != nil {
		t.Fatal("message without an image must not include thumbnail dimensions")
	}
	if message.GetContextInfo() != nil {
		t.Fatal("common link preview must not include context info or external ad reply")
	}
}

func TestBuildLinkExtendedTextMessageRejectsThumbnailWithoutDimensions(t *testing.T) {
	data := &LinkStruct{Text: "Confira https://example.com/article"}

	message := buildLinkExtendedTextMessage(data, "https://example.com/article", []byte{0xff, 0xd8}, 0, 120)

	if message.GetPreviewType() != waE2E.ExtendedTextMessage_PLACEHOLDER {
		t.Fatalf("unexpected preview type: %s", message.GetPreviewType())
	}
	if len(message.GetJPEGThumbnail()) != 0 || message.ThumbnailWidth != nil || message.ThumbnailHeight != nil {
		t.Fatal("thumbnail with invalid dimensions must be discarded")
	}
}
