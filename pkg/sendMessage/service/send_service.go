package send_service

import (
	"bytes"
	"context"
	cryptoRand "crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"

	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "golang.org/x/image/webp"

	config "github.com/EvolutionAPI/evolution-go/pkg/config"
	instance_model "github.com/EvolutionAPI/evolution-go/pkg/instance/model"
	logger_wrapper "github.com/EvolutionAPI/evolution-go/pkg/logger"
	"github.com/EvolutionAPI/evolution-go/pkg/utils"
	whatsmeow_service "github.com/EvolutionAPI/evolution-go/pkg/whatsmeow/service"
	"github.com/chai2010/webp"
	"github.com/gabriel-vasile/mimetype"
	"go.mau.fi/whatsmeow"
	waBinary "go.mau.fi/whatsmeow/binary"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/net/html"
	"google.golang.org/protobuf/proto"
)

type SendService interface {
	SendText(data *TextStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendLink(data *LinkStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendMediaUrl(data *MediaStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendMediaFile(data *MediaStruct, fileData []byte, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendPoll(data *PollStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendSticker(data *StickerStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendLocation(data *LocationStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendContact(data *ContactStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendButton(data *ButtonStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendList(data *ListStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendCarousel(data *CarouselStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendStatusText(data *StatusTextStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendStatusMediaUrl(data *StatusMediaStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	SendStatusMediaFile(data *StatusMediaStruct, fileData []byte, instance *instance_model.Instance) (*MessageSendStruct, error)
}

type sendService struct {
	clientPointer    map[string]*whatsmeow.Client
	whatsmeowService whatsmeow_service.WhatsmeowService
	config           *config.Config
	loggerWrapper    *logger_wrapper.LoggerManager
}

type SendDataStruct struct {
	Id              string
	Number          string
	Delay           int32
	MentionAll      bool
	MentionedJID    []string
	FormatJid       *bool
	Quoted          QuotedStruct
	MediaHandle     string
	AdditionalNodes *[]waBinary.Node
	ForwardingScore *uint32
}

type QuotedStruct struct {
	MessageID   string `json:"messageId"`
	Participant string `json:"participant"`
}

type TextStruct struct {
	Number          string       `json:"number"`
	Text            string       `json:"text"`
	Id              string       `json:"id"`
	Delay           int32        `json:"delay"`
	MentionedJID    []string     `json:"mentionedJid"`
	MentionAll      bool         `json:"mentionAll"`
	FormatJid       *bool        `json:"formatJid,omitempty"`
	Quoted          QuotedStruct `json:"quoted"`
	ForwardingScore *uint32      `json:"forwardingScore,omitempty"`
}

type LinkStruct struct {
	Number       string       `json:"number"`
	Text         string       `json:"text"`
	Title        string       `json:"title"`
	Url          string       `json:"url"`
	Description  string       `json:"description"`
	ImgUrl       string       `json:"imgUrl"`
	Id           string       `json:"id"`
	Delay        int32        `json:"delay"`
	MentionedJID []string     `json:"mentionedJid"`
	MentionAll   bool         `json:"mentionAll"`
	FormatJid    *bool        `json:"formatJid,omitempty"`
	Quoted       QuotedStruct `json:"quoted"`
}

type MediaStruct struct {
	Number          string       `json:"number"`
	Url             string       `json:"url"`
	Type            string       `json:"type"`
	Caption         string       `json:"caption"`
	Filename        string       `json:"filename"`
	Id              string       `json:"id"`
	Delay           int32        `json:"delay"`
	MentionedJID    []string     `json:"mentionedJid"`
	MentionAll      bool         `json:"mentionAll"`
	FormatJid       *bool        `json:"formatJid,omitempty"`
	Quoted          QuotedStruct `json:"quoted"`
	ForwardingScore *uint32      `json:"forwardingScore,omitempty"`
}

type PollStruct struct {
	Id           string       `json:"id"`
	Number       string       `json:"number"`
	Question     string       `json:"question"`
	MaxAnswer    int          `json:"maxAnswer"`
	Options      []string     `json:"options"`
	Delay        int32        `json:"delay"`
	MentionedJID []string     `json:"mentionedJid"`
	MentionAll   bool         `json:"mentionAll"`
	FormatJid    *bool        `json:"formatJid,omitempty"`
	Quoted       QuotedStruct `json:"quoted"`
}

type StickerStruct struct {
	Number       string       `json:"number"`
	Sticker      string       `json:"sticker"`
	Id           string       `json:"id"`
	Delay        int32        `json:"delay"`
	MentionedJID []string     `json:"mentionedJid"`
	MentionAll   bool         `json:"mentionAll"`
	FormatJid    *bool        `json:"formatJid,omitempty"`
	Quoted       QuotedStruct `json:"quoted"`
}

type LocationStruct struct {
	Number       string       `json:"number"`
	Id           string       `json:"id"`
	Name         string       `json:"name"`
	Latitude     float64      `json:"latitude"`
	Longitude    float64      `json:"longitude"`
	Address      string       `json:"address"`
	Delay        int32        `json:"delay"`
	MentionedJID []string     `json:"mentionedJid"`
	MentionAll   bool         `json:"mentionAll"`
	FormatJid    *bool        `json:"formatJid,omitempty"`
	Quoted       QuotedStruct `json:"quoted"`
}

type ContactStruct struct {
	Number       string            `json:"number"`
	Id           string            `json:"id"`
	Vcard        utils.VCardStruct `json:"vcard"`
	Delay        int32             `json:"delay"`
	MentionedJID []string          `json:"mentionedJid"`
	MentionAll   bool              `json:"mentionAll"`
	FormatJid    *bool             `json:"formatJid,omitempty"`
	Quoted       QuotedStruct      `json:"quoted"`
}

type Button struct {
	Type        string `json:"type"`
	DisplayText string `json:"displayText"`
	Id          string `json:"id"`
	CopyCode    string `json:"copyCode"`
	URL         string `json:"url"`
	PhoneNumber string `json:"phoneNumber"`
	Currency    string `json:"currency"`
	Name        string `json:"name"`
	KeyType     string `json:"keyType"`
	Key         string `json:"key"`
}

type ButtonStruct struct {
	Number       string       `json:"number"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Footer       string       `json:"footer"`
	Id           string       `json:"id"`
	Buttons      []Button     `json:"buttons"`
	ImageUrl     string       `json:"imageUrl,omitempty"`
	VideoUrl     string       `json:"videoUrl,omitempty"`
	Delay        int32        `json:"delay"`
	MentionedJID []string     `json:"mentionedJid"`
	MentionAll   bool         `json:"mentionAll"`
	FormatJid    *bool        `json:"formatJid,omitempty"`
	Quoted       QuotedStruct `json:"quoted"`
}

type Row struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	RowId       string `json:"rowId"`
}

type Section struct {
	Title string `json:"title"`
	Rows  []Row  `json:"rows"`
}

type ListStruct struct {
	Number       string       `json:"number"`
	Title        string       `json:"title"`
	Description  string       `json:"description"`
	Id           string       `json:"id"`
	ButtonText   string       `json:"buttonText"`
	FooterText   string       `json:"footerText"`
	Sections     []Section    `json:"sections"`
	Delay        int32        `json:"delay"`
	MentionedJID []string     `json:"mentionedJid"`
	MentionAll   bool         `json:"mentionAll"`
	FormatJid    *bool        `json:"formatJid,omitempty"`
	Quoted       QuotedStruct `json:"quoted"`
}

type CarouselButtonStruct struct {
	Type        string `json:"type"`
	DisplayText string `json:"displayText"`
	Id          string `json:"id"`
	CopyCode    string `json:"copyCode,omitempty"`
}

type CarouselCardHeaderStruct struct {
	Title    string `json:"title"`
	Subtitle string `json:"subtitle"`
	ImageUrl string `json:"imageUrl,omitempty"`
	VideoUrl string `json:"videoUrl,omitempty"`
}

type CarouselCardBodyStruct struct {
	Text string `json:"text"`
}

type CarouselCardStruct struct {
	Header  CarouselCardHeaderStruct `json:"header"`
	Body    CarouselCardBodyStruct   `json:"body"`
	Footer  string                   `json:"footer,omitempty"`
	Buttons []CarouselButtonStruct   `json:"buttons,omitempty"`
}

type CarouselStruct struct {
	Number    string               `json:"number"`
	Body      string               `json:"body,omitempty"`
	Footer    string               `json:"footer,omitempty"`
	Id        string               `json:"id"`
	Delay     int32                `json:"delay"`
	FormatJid *bool                `json:"formatJid,omitempty"`
	Quoted    QuotedStruct         `json:"quoted"`
	Cards     []CarouselCardStruct `json:"cards"`
}

type StatusTextStruct struct {
	Text string `json:"text"`
	Id   string `json:"id"`
}

type StatusMediaStruct struct {
	Type    string `json:"type"`
	Url     string `json:"url"`
	Caption string `json:"caption"`
	Id      string `json:"id"`
}

type MessageSendStruct struct {
	Info               types.MessageInfo
	Message            *waE2E.Message
	MessageContextInfo *waE2E.ContextInfo
}

func (s *sendService) ensureClientConnected(instanceId string) (*whatsmeow.Client, error) {
	client := s.clientPointer[instanceId]
	s.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Checking client connection status - Client exists: %v", instanceId, client != nil)

	if client == nil {
		s.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] No client found, attempting to start new instance", instanceId)
		err := s.whatsmeowService.StartInstance(instanceId)
		if err != nil {
			s.loggerWrapper.GetLogger(instanceId).LogError("[%s] Failed to start instance: %v", instanceId, err)
			return nil, errors.New("no active session found")
		}

		s.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Instance started, waiting 2 seconds...", instanceId)
		time.Sleep(2 * time.Second)

		client = s.clientPointer[instanceId]
		s.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Checking new client - Exists: %v, Connected: %v",
			instanceId,
			client != nil,
			client != nil && client.IsConnected())

		if client == nil || !client.IsConnected() {
			s.loggerWrapper.GetLogger(instanceId).LogError("[%s] New client validation failed - Exists: %v, Connected: %v",
				instanceId,
				client != nil,
				client != nil && client.IsConnected())
			return nil, errors.New("no active session found")
		}
	} else if !client.IsConnected() {
		s.loggerWrapper.GetLogger(instanceId).LogError("[%s] Existing client is disconnected - Connected status: %v",
			instanceId,
			client.IsConnected())
		return nil, errors.New("client disconnected")
	}

	s.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Client successfully validated - Connected: %v", instanceId, client.IsConnected())
	return client, nil
}

// ensureClientConnectedWithRetry attempts to ensure client connection with automatic reconnection and retry
func (s *sendService) ensureClientConnectedWithRetry(instanceId string, maxRetries int) (*whatsmeow.Client, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Connection attempt %d/%d", instanceId, attempt, maxRetries)

		client, err := s.ensureClientConnected(instanceId)
		if err == nil {
			return client, nil
		}

		// Check if it's a disconnection error that we can retry
		if err.Error() == "client disconnected" || err.Error() == "no active session found" {
			s.loggerWrapper.GetLogger(instanceId).LogWarn("[%s] Client disconnected on attempt %d/%d, attempting reconnection...", instanceId, attempt, maxRetries)

			// Attempt to reconnect the client
			reconnectErr := s.whatsmeowService.ReconnectClient(instanceId)
			if reconnectErr != nil {
				s.loggerWrapper.GetLogger(instanceId).LogError("[%s] Failed to reconnect client on attempt %d: %v", instanceId, attempt, reconnectErr)
			} else {
				s.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Reconnection initiated on attempt %d, waiting 3 seconds...", instanceId, attempt)
				time.Sleep(3 * time.Second)
			}

			// If this is not the last attempt, continue to retry
			if attempt < maxRetries {
				waitTime := time.Duration(attempt*2) * time.Second // Progressive backoff
				s.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Waiting %v before retry attempt %d", instanceId, waitTime, attempt+1)
				time.Sleep(waitTime)
				continue
			}
		}

		// If it's the last attempt or a non-retryable error, return the error
		s.loggerWrapper.GetLogger(instanceId).LogError("[%s] Failed to ensure client connection after %d attempts: %v", instanceId, attempt, err)
		return nil, err
	}

	return nil, fmt.Errorf("failed to connect client after %d attempts", maxRetries)
}

func validateMessageFields(phone string, formatJid *bool, messageID *string, participant *string) (types.JID, error) {
	// Apply formatting if formatJid is true (default)
	shouldFormat := true // Default value
	if formatJid != nil {
		shouldFormat = *formatJid
	}

	var finalPhone string
	if shouldFormat {
		// Extract raw number if it's already a JID, then apply CreateJID formatting
		rawNumber := phone
		if strings.Contains(phone, "@s.whatsapp.net") {
			rawNumber = strings.Split(phone, "@")[0]
		}

		normalizedJID, err := utils.CreateJID(rawNumber)
		if err != nil {
			// If CreateJID fails, try with ParseJID as fallback
			recipient, ok := utils.ParseJID(phone)
			if !ok {
				return types.NewJID("", types.DefaultUserServer), fmt.Errorf("could not parse phone: %s", phone)
			}
			finalPhone = recipient.String()
		} else {
			finalPhone = normalizedJID
		}
	} else {
		// Use phone as received without formatting
		finalPhone = phone
	}

	recipient, ok := utils.ParseJID(finalPhone)
	if !ok {
		return types.NewJID("", types.DefaultUserServer), errors.New("could not parse formatted phone")
	}

	if messageID != nil {
		if participant == nil {
			return types.NewJID("", types.DefaultUserServer), errors.New("missing Participant in ContextInfo")
		}
	}

	if participant != nil {
		if messageID == nil {
			return types.NewJID("", types.DefaultUserServer), errors.New("missing StanzaId in ContextInfo")
		}
	}

	return recipient, nil
}

// validateAndCheckUserExists validates message fields and checks if the user exists on WhatsApp
// Now uses the new approach: CheckUser with formatJid=false by default, and uses remoteJID for messaging
func (s *sendService) validateAndCheckUserExists(phone string, formatJid *bool, messageID *string, participant *string, instance *instance_model.Instance) (types.JID, error) {
	// Skip WhatsApp check if disabled in config
	if !s.config.CheckUserExists {
		s.loggerWrapper.GetLogger(instance.Id).LogDebug("[%s] User existence check disabled by configuration", instance.Id)
		// Use original validation logic when check is disabled
		return validateMessageFields(phone, formatJid, messageID, participant)
	}

	// Skip WhatsApp check for group messages, broadcast, newsletter, and LID
	if strings.Contains(phone, "@g.us") || strings.Contains(phone, "@broadcast") || strings.Contains(phone, "@newsletter") || strings.Contains(phone, "@lid") {
		return validateMessageFields(phone, formatJid, messageID, participant)
	}

	// Get the client to check if user exists on WhatsApp
	client, err := s.ensureClientConnected(instance.Id)
	if err != nil {
		return types.NewJID("", types.DefaultUserServer), fmt.Errorf("failed to connect client: %v", err)
	}

	// Use CheckUser approach: formatJid=false by default
	formatJidForCheck := false

	// First attempt with formatJid=false
	remoteJID, found, err := s.checkSingleUserExists(client, phone, formatJidForCheck, instance.Id)
	if err != nil {
		s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] Failed to check user existence: %v", instance.Id, err)
		// Continue with sending even if check fails (network issues, etc.)
		return validateMessageFields(phone, formatJid, messageID, participant)
	}

	// If not found with formatJid=false, try with formatJid=true as fallback
	if !found {
		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] User not found with formatJid=false, trying with formatJid=true", instance.Id)
		remoteJIDRetry, foundRetry, errRetry := s.checkSingleUserExists(client, phone, true, instance.Id)
		if errRetry == nil && foundRetry {
			remoteJID = remoteJIDRetry
			found = foundRetry
		}
	}

	if !found {
		return types.NewJID("", types.DefaultUserServer), fmt.Errorf("number %s is not registered on WhatsApp", phone)
	}

	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Number %s verified as valid WhatsApp user, using remoteJID: %s", instance.Id, phone, remoteJID)

	// Validate the remoteJID with formatJid=false for message sending
	formatJidFalse := false
	return validateMessageFields(remoteJID, &formatJidFalse, messageID, participant)
}

// checkSingleUserExists checks if a single user exists on WhatsApp with the specified formatJid setting
// Returns: remoteJID, found, error
func (s *sendService) checkSingleUserExists(client *whatsmeow.Client, phone string, formatJid bool, instanceId string) (string, bool, error) {
	phoneNumbers, err := utils.PrepareNumbersForWhatsAppCheck([]string{phone}, &formatJid)
	if err != nil {
		return "", false, fmt.Errorf("failed to prepare number for WhatsApp check: %v", err)
	}

	// Check if the number exists on WhatsApp
	resp, err := client.IsOnWhatsApp(context.Background(), phoneNumbers)
	if err != nil {
		return "", false, fmt.Errorf("failed to check if number %s exists on WhatsApp: %v", phoneNumbers[0], err)
	}

	// Verify if the number was found
	if len(resp) == 0 {
		return "", false, fmt.Errorf("number %s not found in WhatsApp response", phoneNumbers[0])
	}

	// Check if the first result indicates the number is on WhatsApp
	if !resp[0].IsIn {
		return "", false, nil // Not an error, just not found
	}

	// Return the remoteJID from WhatsApp's response
	remoteJID := fmt.Sprintf("%v", resp[0].JID)
	return remoteJID, true, nil
}

func findURL(text string) string {
	urlRegex := `http[s]?://(?:[a-zA-Z]|[0-9]|[$-_@.&+]|[!*\\(\\),]|(?:%[0-9a-fA-F][0-9a-fA-F]))+`
	re := regexp.MustCompile(urlRegex)
	urls := re.FindAllString(text, -1)
	if len(urls) > 0 {
		return urls[0]
	}
	return ""
}

func (s *sendService) SendText(data *TextStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	return s.sendTextWithRetry(data, instance, 3) // 3 tentativas máximas
}

func (s *sendService) sendTextWithRetry(data *TextStruct, instance *instance_model.Instance, maxRetries int) (*MessageSendStruct, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendText attempt %d/%d", instance.Id, attempt, maxRetries)

		_, err := s.ensureClientConnectedWithRetry(instance.Id, 2)
		if err != nil {
			if attempt == maxRetries {
				return nil, err
			}
			continue
		}

		msg := &waE2E.Message{
			ExtendedTextMessage: &waE2E.ExtendedTextMessage{
				Text: &data.Text,
			},
		}

		message, err := s.SendMessage(instance, msg, "ExtendedTextMessage", &SendDataStruct{
			Id:              data.Id,
			Number:          data.Number,
			Quoted:          data.Quoted,
			Delay:           data.Delay,
			MentionAll:      data.MentionAll,
			MentionedJID:    data.MentionedJID,
			FormatJid:       data.FormatJid,
			ForwardingScore: data.ForwardingScore,
		})

		if err != nil {
			// Check if it's a client disconnection error
			if strings.Contains(err.Error(), "client disconnected") || strings.Contains(err.Error(), "no active session") {
				s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] SendText failed due to disconnection on attempt %d/%d: %v", instance.Id, attempt, maxRetries, err)
				if attempt < maxRetries {
					waitTime := time.Duration(attempt) * time.Second
					s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Waiting %v before retry", instance.Id, waitTime)
					time.Sleep(waitTime)
					continue
				}
			}
			return nil, err
		}

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendText successful on attempt %d", instance.Id, attempt)
		return message, nil
	}

	return nil, fmt.Errorf("failed to send text after %d attempts", maxRetries)
}

func fetchLinkMetadata(targetUrl string) (string, string, string, error) {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", targetUrl, nil)
	if err != nil {
		return "", "", "", err
	}

	// Identificar-se como WhatsApp para obter metadados otimizados
	req.Header.Set("User-Agent", "WhatsApp/2.24.8.85")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", "", fmt.Errorf("failed to fetch metadata: status %d", resp.StatusCode)
	}

	doc, err := html.Parse(resp.Body)
	if err != nil {
		return "", "", "", err
	}

	var title, description, imgURL string

	var f func(*html.Node)
	f = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "title" && n.FirstChild != nil && title == "" {
				title = n.FirstChild.Data
			}
			if n.Data == "meta" {
				var property, name, itemprop, content string
				for _, attr := range n.Attr {
					key := strings.ToLower(strings.TrimSpace(attr.Key))
					val := strings.TrimSpace(attr.Val)
					if key == "property" {
						property = strings.ToLower(val)
					} else if key == "name" {
						name = strings.ToLower(val)
					} else if key == "itemprop" {
						itemprop = strings.ToLower(val)
					} else if key == "content" {
						content = val
					}
				}

				// Prioridade para OpenGraph tags
				if property == "og:title" && content != "" {
					title = content
				}
				if (property == "og:description" || name == "description") && description == "" {
					description = content
				}
				if (property == "og:image" ||
					property == "og:image:url" ||
					property == "og:image:secure_url" ||
					name == "twitter:image" ||
					name == "twitter:image:src" ||
					itemprop == "image") && imgURL == "" {
					imgURL = content
				}
			}
			if n.Data == "link" && imgURL == "" {
				var rel, as, href string
				for _, attr := range n.Attr {
					key := strings.ToLower(strings.TrimSpace(attr.Key))
					val := strings.TrimSpace(attr.Val)
					if key == "rel" {
						rel = strings.ToLower(val)
					} else if key == "as" {
						as = strings.ToLower(val)
					} else if key == "href" {
						href = val
					}
				}
				if (strings.Contains(rel, "image_src") ||
					(strings.Contains(rel, "preload") && as == "image")) &&
					isUsablePreviewImageURL(href) {
					imgURL = href
				}
			}
			if n.Data == "img" && imgURL == "" {
				var src, srcset string
				for _, attr := range n.Attr {
					key := strings.ToLower(strings.TrimSpace(attr.Key))
					val := strings.TrimSpace(attr.Val)
					if (key == "src" || key == "data-src" || key == "data-lazy-src") && src == "" {
						src = val
					} else if (key == "srcset" || key == "data-srcset" || key == "data-lazy-srcset") && srcset == "" {
						srcset = val
					}
				}
				if srcsetURL := firstSrcsetURL(srcset); srcsetURL != "" {
					imgURL = srcsetURL
				} else if isUsablePreviewImageURL(src) {
					imgURL = src
				}
			}
			if n.Data == "script" && imgURL == "" {
				var scriptType string
				for _, attr := range n.Attr {
					if strings.ToLower(strings.TrimSpace(attr.Key)) == "type" {
						scriptType = strings.ToLower(strings.TrimSpace(attr.Val))
					}
				}
				if strings.Contains(scriptType, "ld+json") {
					var payload interface{}
					if err := json.Unmarshal([]byte(strings.TrimSpace(nodeText(n))), &payload); err == nil {
						imgURL = findJSONLDImage(payload)
					}
				}
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			f(c)
		}
	}

	f(doc)

	// Resolver URLs relativas de imagem
	if imgURL != "" {
		if base, err := url.Parse(targetUrl); err == nil {
			if img, err := url.Parse(imgURL); err == nil {
				imgURL = base.ResolveReference(img).String()
			}
		}
	}

	return strings.TrimSpace(title), strings.TrimSpace(description), imgURL, nil
}

func findJSONLDImage(value interface{}) string {
	switch v := value.(type) {
	case string:
		if isUsablePreviewImageURL(v) {
			return strings.TrimSpace(v)
		}
	case []interface{}:
		for _, item := range v {
			if imageURL := findJSONLDImage(item); imageURL != "" {
				return imageURL
			}
		}
	case map[string]interface{}:
		for _, key := range []string{"image", "thumbnailUrl", "thumbnail", "primaryImageOfPage"} {
			if imageURL := findJSONLDImage(v[key]); imageURL != "" {
				return imageURL
			}
		}
		if jsonLDMapLooksLikeImage(v) {
			for _, key := range []string{"url", "contentUrl"} {
				if imageURL := findJSONLDImage(v[key]); imageURL != "" {
					return imageURL
				}
			}
		}
		for _, item := range v {
			switch item.(type) {
			case []interface{}, map[string]interface{}:
				if imageURL := findJSONLDImage(item); imageURL != "" {
					return imageURL
				}
			}
		}
	}

	return ""
}

func jsonLDMapLooksLikeImage(value map[string]interface{}) bool {
	for _, key := range []string{"contentUrl", "thumbnailUrl"} {
		if _, ok := value[key]; ok {
			return true
		}
	}

	rawType, ok := value["@type"]
	if !ok {
		return false
	}
	switch typed := rawType.(type) {
	case string:
		return strings.Contains(strings.ToLower(typed), "image")
	case []interface{}:
		for _, item := range typed {
			if typeName, ok := item.(string); ok && strings.Contains(strings.ToLower(typeName), "image") {
				return true
			}
		}
	}

	return false
}

func nodeText(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}

	var builder strings.Builder
	for child := n.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(nodeText(child))
	}
	return builder.String()
}

func firstSrcsetURL(srcset string) string {
	var selected string
	for _, candidate := range strings.Split(srcset, ",") {
		fields := strings.Fields(strings.TrimSpace(candidate))
		if len(fields) > 0 && isUsablePreviewImageURL(fields[0]) {
			selected = fields[0]
		}
	}
	return selected
}

func isUsablePreviewImageURL(imageURL string) bool {
	imageURL = strings.TrimSpace(imageURL)
	lowerURL := strings.ToLower(imageURL)
	if imageURL == "" ||
		strings.HasPrefix(lowerURL, "data:") ||
		strings.HasPrefix(lowerURL, "blob:") ||
		strings.Contains(lowerURL, "favicon") ||
		strings.Contains(lowerURL, "apple-touch-icon") ||
		strings.HasSuffix(lowerURL, ".svg") {
		return false
	}
	return true
}

func (s *sendService) resizeThumbnail(data []byte, maxDim int) []byte {
	if len(data) == 0 {
		return nil
	}

	// Tentar decodificar a imagem (suporta JPEG, PNG, WEBP via drivers registrados)
	src, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		s.loggerWrapper.GetLogger("").LogWarn("Failed to decode image for thumbnail: %v", err)
		return data
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()

	// Se já estiver dentro do limite, não redimensionar
	if w <= maxDim && h <= maxDim && len(data) < 32768 {
		return data
	}

	// Calcular novas dimensões mantendo o aspect ratio
	newW, newH := w, h
	if w > maxDim || h > maxDim {
		if w > h {
			newW = maxDim
			newH = (h * maxDim) / w
		} else {
			newH = maxDim
			newW = (w * maxDim) / h
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	xdraw.BiLinear.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	var buf bytes.Buffer
	// Codificar como JPEG com qualidade 80 para garantir tamanho pequeno
	err = jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 80})
	if err != nil {
		return data
	}

	s.loggerWrapper.GetLogger("").LogInfo("Thumbnail resized from %d to %d bytes (format: %s)", len(data), buf.Len(), format)
	return buf.Bytes()
}

// makePDFThumbnail rasterizes the first page of a PDF into a JPEG thumbnail
// using the external "pdftoppm" tool (poppler-utils). It returns nil when
// pdftoppm is not installed or rasterization fails, so callers can gracefully
// send the document without a preview instead of failing the request.
func (s *sendService) makePDFThumbnail(fileData []byte, maxWidth int) []byte {
	if _, err := exec.LookPath("pdftoppm"); err != nil {
		return nil
	}

	scaleWidth := maxWidth
	if scaleWidth < 1 {
		scaleWidth = 72
	}

	// Render only the first page to a PNG on stdout, scaled to scaleWidth.
	// "-scale-to-y -1" keeps the original aspect ratio.
	cmd := exec.Command("pdftoppm",
		"-png",
		"-f", "1",
		"-l", "1",
		"-singlefile",
		"-scale-to-x", strconv.Itoa(scaleWidth),
		"-scale-to-y", "-1",
	)
	cmd.Stdin = bytes.NewReader(fileData)

	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		s.loggerWrapper.GetLogger("").LogWarn("Failed to rasterize PDF thumbnail with pdftoppm: %v", err)
		return nil
	}
	if out.Len() == 0 {
		return nil
	}

	// Re-encode the rendered PNG as a JPEG thumbnail via the existing local helper.
	return s.resizeThumbnail(out.Bytes(), maxWidth)
}

func normalizeLinkPreviewThumbnail(data []byte, maxDim int) ([]byte, uint32, uint32, string, error) {
	if len(data) == 0 {
		return nil, 0, 0, "", nil
	}
	if maxDim < 1 {
		maxDim = 1
	}

	src, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, 0, 0, "", fmt.Errorf("decode link preview thumbnail: %w", err)
	}

	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w <= 0 || h <= 0 {
		return nil, 0, 0, format, fmt.Errorf("invalid link preview thumbnail dimensions: %dx%d", w, h)
	}

	newW, newH := w, h
	if w > maxDim || h > maxDim {
		if w > h {
			newW = maxDim
			newH = max(1, (h*maxDim)/w)
		} else {
			newH = maxDim
			newW = max(1, (w*maxDim)/h)
		}
	}

	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	xdraw.BiLinear.Scale(dst, dst.Bounds(), src, bounds, draw.Over, nil)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 86}); err != nil {
		return nil, 0, 0, format, fmt.Errorf("encode link preview thumbnail as JPEG: %w", err)
	}

	return buf.Bytes(), uint32(newW), uint32(newH), format, nil
}

func (s *sendService) buildLinkPreviewThumbnail(data []byte, maxDim int) ([]byte, uint32, uint32) {
	thumbnail, width, height, format, err := normalizeLinkPreviewThumbnail(data, maxDim)
	if err != nil {
		s.loggerWrapper.GetLogger("").LogWarn("Discarding invalid link preview thumbnail: %v", err)
		return nil, 0, 0
	}
	if thumbnail == nil {
		return nil, 0, 0
	}

	s.loggerWrapper.GetLogger("").LogInfo("Link preview thumbnail normalized from %d to %d bytes (format: %s)", len(data), len(thumbnail), format)
	return thumbnail, width, height
}

func (s *sendService) getVideoThumbnail(data []byte) []byte {
	if len(data) == 0 {
		return nil
	}

	// Tentar encontrar ffmpeg
	ffmpegPath, err := exec.LookPath("ffmpeg")
	if err != nil {
		s.loggerWrapper.GetLogger("").LogWarn("ffmpeg not found, skipping video thumbnail generation")
		return nil
	}

	// Criar arquivo temporário para o vídeo
	tmpFile, err := os.CreateTemp("", "video_*.mp4")
	if err != nil {
		return nil
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	if _, err := tmpFile.Write(data); err != nil {
		return nil
	}

	// Executar ffmpeg para extrair o primeiro quadro (1 segundo adentro para evitar telas pretas)
	// -vframes 1: extrai apenas 1 frame
	// -f image2 pipe:1: envia o output (imagem) para o stdout
	cmd := exec.Command(ffmpegPath, "-i", tmpFile.Name(), "-ss", "00:00:01", "-vframes", "1", "-f", "image2", "pipe:1")
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		// Se falhar em 1 segundo, tentar no início (0 segundos)
		cmd = exec.Command(ffmpegPath, "-i", tmpFile.Name(), "-vframes", "1", "-f", "image2", "pipe:1")
		out.Reset()
		cmd.Stdout = &out
		if err := cmd.Run(); err != nil {
			return nil
		}
	}

	// Redimensionar o frame extraído para servir de thumbnail
	return s.resizeThumbnail(out.Bytes(), 100)
}

func (s *sendService) SendLink(data *LinkStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	return s.sendLinkWithRetry(data, instance, 3)
}

func buildLinkExtendedTextMessage(data *LinkStruct, matchedText string, thumbnail []byte, thumbnailWidth, thumbnailHeight uint32) *waE2E.ExtendedTextMessage {
	previewType := waE2E.ExtendedTextMessage_PLACEHOLDER
	if len(thumbnail) > 0 && thumbnailWidth > 0 && thumbnailHeight > 0 {
		previewType = waE2E.ExtendedTextMessage_IMAGE
	} else {
		thumbnail = nil
	}

	extendedText := &waE2E.ExtendedTextMessage{
		Text:          proto.String(data.Text),
		Title:         proto.String(data.Title),
		MatchedText:   proto.String(matchedText),
		PreviewType:   &previewType,
		JPEGThumbnail: thumbnail,
		Description:   proto.String(data.Description),
	}
	if len(thumbnail) > 0 {
		extendedText.ThumbnailWidth = proto.Uint32(thumbnailWidth)
		extendedText.ThumbnailHeight = proto.Uint32(thumbnailHeight)
	}

	return extendedText
}

func (s *sendService) sendLinkWithRetry(data *LinkStruct, instance *instance_model.Instance, maxRetries int) (*MessageSendStruct, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendLink attempt %d/%d", instance.Id, attempt, maxRetries)

		_, err := s.ensureClientConnectedWithRetry(instance.Id, 2)
		if err != nil {
			if attempt == maxRetries {
				return nil, err
			}
			continue
		}

		matchedText := data.Url
		if matchedText == "" {
			matchedText = findURL(data.Text)
		}

		if matchedText != "" {
			// Só buscar metadados se os campos estiverem vazios
			if data.Title == "" || data.Description == "" || data.ImgUrl == "" {
				title, description, imgUrl, err := fetchLinkMetadata(matchedText)
				if err == nil {
					if data.Title == "" {
						data.Title = title
					}
					if data.Description == "" {
						data.Description = description
					}
					if data.ImgUrl == "" {
						data.ImgUrl = imgUrl
					}
					s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Link metadata fetched: url=%s title=%q descriptionLen=%d imgUrl=%q", instance.Id, matchedText, data.Title, len(data.Description), data.ImgUrl)
				} else {
					s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] Failed to fetch link metadata for %s: %v", instance.Id, matchedText, err)
				}
			}
		}

		var fileData []byte
		var thumbnailWidth, thumbnailHeight uint32
		if data.ImgUrl != "" {
			s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Downloading link preview image: %s", instance.Id, data.ImgUrl)
			// Download da imagem da miniatura com User-Agent para evitar bloqueios em produção
			imgReq, err := http.NewRequest("GET", data.ImgUrl, nil)
			if err == nil {
				imgReq.Header.Set("User-Agent", "WhatsApp/2.24.8.85")
				imgReq.Header.Set("Accept", "image/webp,image/apng,image/svg+xml,image/*,*/*;q=0.8")

				imgClient := &http.Client{Timeout: 10 * time.Second}
				imgResp, err := imgClient.Do(imgReq)
				if err == nil && imgResp.StatusCode == http.StatusOK {
					defer imgResp.Body.Close()
					rawImageData, _ := io.ReadAll(imgResp.Body)
					s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Link preview image downloaded: contentType=%q bytes=%d", instance.Id, imgResp.Header.Get("Content-Type"), len(rawImageData))
					fileData, thumbnailWidth, thumbnailHeight = s.buildLinkPreviewThumbnail(rawImageData, 200)
					if fileData == nil {
						s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] Link preview image ignored because it could not be converted to JPEG thumbnail: %s", instance.Id, data.ImgUrl)
					}
				} else if err != nil {
					s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] Failed to download link preview image: %v", instance.Id, err)
				} else {
					s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] Failed to download link preview image: status %d", instance.Id, imgResp.StatusCode)
				}
			} else {
				s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] Invalid link preview image URL %q: %v", instance.Id, data.ImgUrl, err)
			}
		} else {
			s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] Link preview image URL is empty after metadata fetch for %s", instance.Id, matchedText)
		}

		msg := &waE2E.Message{
			ExtendedTextMessage: buildLinkExtendedTextMessage(data, matchedText, fileData, thumbnailWidth, thumbnailHeight),
		}

		message, err := s.SendMessage(instance, msg, "ExtendedTextMessage", &SendDataStruct{
			Id:           data.Id,
			Number:       data.Number,
			Quoted:       data.Quoted,
			Delay:        data.Delay,
			MentionAll:   data.MentionAll,
			MentionedJID: data.MentionedJID,
			FormatJid:    data.FormatJid,
		})

		if err != nil {
			// Check if it's a client disconnection error
			if strings.Contains(err.Error(), "client disconnected") || strings.Contains(err.Error(), "no active session") {
				s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] SendLink failed due to disconnection on attempt %d/%d: %v", instance.Id, attempt, maxRetries, err)
				if attempt < maxRetries {
					waitTime := time.Duration(attempt) * time.Second
					s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Waiting %v before retry", instance.Id, waitTime)
					time.Sleep(waitTime)
					continue
				}
			}
			return nil, err
		}

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendLink successful on attempt %d", instance.Id, attempt)
		return message, nil
	}

	return nil, fmt.Errorf("failed to send link after %d attempts", maxRetries)
}

type ConvertAudio struct {
	Url    string `json:"url,omitempty"`
	Base64 string `json:"base64,omitempty"`
}

type ApiResponse struct {
	Duration int    `json:"duration"`
	Audio    string `json:"audio"`
}

func convertAudioWithApi(apiUrl string, apiKey string, convertData ConvertAudio) ([]byte, int, error) {
	var requestBody bytes.Buffer
	writer := multipart.NewWriter(&requestBody)

	// Adiciona o campo "url" ao form-data se a URL for fornecida
	if convertData.Url != "" {
		err := writer.WriteField("url", convertData.Url)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao adicionar a URL no form-data: %v", err)
		}
	}

	// Adiciona o campo "base64" ao form-data se a string base64 for fornecida
	if convertData.Base64 != "" {
		err := writer.WriteField("base64", convertData.Base64)
		if err != nil {
			return nil, 0, fmt.Errorf("erro ao adicionar o base64 no form-data: %v", err)
		}
	}

	// Fecha o writer multipart
	err := writer.Close()
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao finalizar o form-data: %v", err)
	}

	req, err := http.NewRequest("POST", apiUrl, &requestBody)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao criar a requisição: %v", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("apikey", apiKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao enviar a requisição: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao ler a resposta: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("requisição falhou com status: %d, resposta: %s", resp.StatusCode, string(body))
	}

	var apiResponse ApiResponse
	err = json.Unmarshal(body, &apiResponse)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao deserializar a resposta: %v", err)
	}

	base64ToBytes, err := base64.StdEncoding.DecodeString(apiResponse.Audio)
	if err != nil {
		return nil, 0, fmt.Errorf("erro ao decodificar o áudio: %v", err)
	}

	return base64ToBytes, apiResponse.Duration, nil
}

func convertAudioToOpusWithDuration(inputData []byte) ([]byte, int, error) {
	cmd := exec.Command("ffmpeg", "-i", "pipe:0",
		"-f",
		"ogg",
		"-vn",
		"-c:a",
		"libopus",
		"-avoid_negative_ts",
		"make_zero",
		"-b:a",
		"128k",
		"-ar",
		"48000",
		"-ac",
		"1",
		"-write_xing",
		"0",
		"-compression_level",
		"10",
		"-application",
		"voip",
		"-fflags",
		"+bitexact",
		"-flags",
		"+bitexact",
		"-id3v2_version",
		"0",
		"-map_metadata",
		"-1",
		"-map_chapters",
		"-1",
		"-write_bext",
		"0",
		"pipe:1",
	)

	var outBuffer bytes.Buffer
	var errBuffer bytes.Buffer

	cmd.Stdin = bytes.NewReader(inputData)
	cmd.Stdout = &outBuffer
	cmd.Stderr = &errBuffer

	err := cmd.Run()
	if err != nil {
		return nil, 0, fmt.Errorf("error during conversion: %v, details: %s", err, errBuffer.String())
	}

	convertedData := outBuffer.Bytes()

	outputText := errBuffer.String()

	splitTime := strings.Split(outputText, "time=")

	if len(splitTime) < 2 {
		return nil, 0, errors.New("duração não encontrada")
	}

	// Use the last occurrence of time= in case there are multiple
	timeString := splitTime[len(splitTime)-1]

	re := regexp.MustCompile(`(\d+):(\d+):(\d+\.\d+)`)
	matches := re.FindStringSubmatch(timeString)
	if len(matches) != 4 {
		return nil, 0, errors.New("formato de duração não encontrado")
	}

	hours, _ := strconv.ParseFloat(matches[1], 64)
	minutes, _ := strconv.ParseFloat(matches[2], 64)
	seconds, _ := strconv.ParseFloat(matches[3], 64)
	duration := int(hours*3600 + minutes*60 + seconds)

	return convertedData, duration, nil
}

func (s *sendService) SendMediaFile(data *MediaStruct, fileData []byte, instance *instance_model.Instance) (*MessageSendStruct, error) {
	return s.sendMediaFileWithRetry(data, fileData, instance, 3)
}

func (s *sendService) sendMediaFileWithRetry(data *MediaStruct, fileData []byte, instance *instance_model.Instance, maxRetries int) (*MessageSendStruct, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendMediaFile attempt %d/%d", instance.Id, attempt, maxRetries)

		client, err := s.ensureClientConnectedWithRetry(instance.Id, 2)
		if err != nil {
			if attempt == maxRetries {
				return nil, err
			}
			continue
		}

		mime, _ := mimetype.DetectReader(bytes.NewReader(fileData))
		mimeType := mime.String()

		var uploadType whatsmeow.MediaType
		var duration int

		switch data.Type {
		case "image":
			if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/webp" {
				errMsg := fmt.Sprintf("Invalid file format: '%s'. Only 'image/jpeg', 'image/png' and 'image/webp' are accepted", mimeType)
				return nil, errors.New(errMsg)
			}
			if mimeType == "image/webp" {
				mimeType = "image/jpeg"
			}
			uploadType = whatsmeow.MediaImage
		case "video":
			if mimeType != "video/mp4" {
				errMsg := fmt.Sprintf("Invalid file format: '%s'. Only 'video/mp4' is accepted", mimeType)
				return nil, errors.New(errMsg)
			}
			uploadType = whatsmeow.MediaVideo
		case "audio":
			converterApiUrl := s.config.ApiAudioConverter
			converterApiKey := s.config.ApiAudioConverterKey
			var convertedData []byte
			var err error
			if converterApiUrl == "" {

				convertedData, duration, err = convertAudioToOpusWithDuration(fileData)
				if err != nil {
					return nil, err
				}
			} else {
				convertedData, duration, err = convertAudioWithApi(converterApiUrl, converterApiKey, ConvertAudio{Base64: base64.StdEncoding.EncodeToString(fileData)})
				if err != nil {
					return nil, err
				}
			}

			fileData = convertedData
			mimeType = "audio/ogg; codecs=opus"
			uploadType = whatsmeow.MediaAudio
		case "document":
			uploadType = whatsmeow.MediaDocument
		default:
			return nil, errors.New("invalid media type")
		}

		// Detectar se é newsletter para usar upload sem criptografia
		isNewsletter := strings.Contains(data.Number, "@newsletter")

		// Validar se é documento em newsletter (não suportado)
		if isNewsletter && data.Type == "document" {
			return nil, errors.New("documentos não são suportados em canais do WhatsApp. Use imagem, vídeo, áudio ou enquete")
		}

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendMediaFile - Upload iniciado (Newsletter: %v)...", instance.Id, isNewsletter)

		var uploaded whatsmeow.UploadResponse
		if isNewsletter {
			// Newsletter: upload SEM criptografia
			uploaded, err = client.UploadNewsletter(context.Background(), fileData, uploadType)
			s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Newsletter upload - Handle: %s", instance.Id, uploaded.Handle)
		} else {
			// Normal: upload COM criptografia
			uploaded, err = client.Upload(context.Background(), fileData, uploadType)
		}

		if err != nil {
			return nil, err
		}

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Media uploaded with size %d", instance.Id, uploaded.FileLength)

		var media *waE2E.Message
		var mediaType string

		switch data.Type {
		case "image":
			if isNewsletter {
				media = &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
					Caption:    proto.String(data.Caption),
					URL:        &uploaded.URL,
					DirectPath: &uploaded.DirectPath,
					Mimetype:   proto.String(mimeType),
					FileSHA256: uploaded.FileSHA256,
					FileLength: &uploaded.FileLength,
				}}
			} else {
				thumbnail := s.resizeThumbnail(fileData, 100)
				media = &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
					Caption:       proto.String(data.Caption),
					URL:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String(mimeType),
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(fileData))),
					JPEGThumbnail: thumbnail,
				}}
			}
			mediaType = "ImageMessage"
		case "video":
			if isNewsletter {
				media = &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
					Caption:    proto.String(data.Caption),
					URL:        &uploaded.URL,
					DirectPath: &uploaded.DirectPath,
					Mimetype:   proto.String(mimeType),
					FileSHA256: uploaded.FileSHA256,
					FileLength: &uploaded.FileLength,
				}}
			} else {
				thumbnail := s.getVideoThumbnail(fileData)
				media = &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
					Caption:       proto.String(data.Caption),
					URL:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String(mimeType),
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(fileData))),
					JPEGThumbnail: thumbnail,
				}}
			}
			mediaType = "VideoMessage"
		case "ptv":
			if isNewsletter {
				media = &waE2E.Message{PtvMessage: &waE2E.VideoMessage{
					URL:        &uploaded.URL,
					DirectPath: &uploaded.DirectPath,
					Mimetype:   proto.String(mimeType),
					FileSHA256: uploaded.FileSHA256,
					FileLength: &uploaded.FileLength,
				}}
			} else {
				media = &waE2E.Message{PtvMessage: &waE2E.VideoMessage{
					URL:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String(mimeType),
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(fileData))),
				}}
			}
			mediaType = "PtvMessage"
		case "audio":
			if isNewsletter {
				media = &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
					URL:        &uploaded.URL,
					PTT:        proto.Bool(true),
					DirectPath: &uploaded.DirectPath,
					Mimetype:   proto.String(mimeType),
					FileSHA256: uploaded.FileSHA256,
					FileLength: &uploaded.FileLength,
					Seconds:    proto.Uint32(uint32(duration)),
				}}
			} else {
				media = &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
					URL:           proto.String(uploaded.URL),
					PTT:           proto.Bool(true),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String(mimeType),
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uploaded.FileLength),
					Seconds:       proto.Uint32(uint32(duration)),
				}}
			}
			mediaType = "AudioMessage"
		case "document":
			// For PDF documents, rasterize page 1 into a JPEG preview thumbnail.
			// A missing pdftoppm or a failure yields nil and the document is
			// sent without a preview instead of failing the request.
			var jpegThumb []byte
			if mimeType == "application/pdf" {
				jpegThumb = s.makePDFThumbnail(fileData, 200)
			}
			if isNewsletter {
				media = &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
					FileName:      &data.Filename,
					Caption:       proto.String(data.Caption),
					URL:           &uploaded.URL,
					DirectPath:    &uploaded.DirectPath,
					Mimetype:      proto.String(mimeType),
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    &uploaded.FileLength,
					JPEGThumbnail: jpegThumb,
				}}
			} else {
				media = &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
					FileName:      &data.Filename,
					Caption:       proto.String(data.Caption),
					URL:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String(mimeType),
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(fileData))),
					JPEGThumbnail: jpegThumb,
				}}
			}

			if media.GetDocumentMessage().GetCaption() != "" {
				media.DocumentWithCaptionMessage = &waE2E.FutureProofMessage{
					Message: &waE2E.Message{
						DocumentMessage: media.DocumentMessage,
					},
				}
				media.DocumentMessage = nil
			}

			mediaType = "DocumentMessage"
		default:
			return nil, errors.New("invalid media type")
		}

		message, err := s.SendMessage(instance, media, mediaType, &SendDataStruct{
			Id:              data.Id,
			Number:          data.Number,
			Quoted:          data.Quoted,
			Delay:           data.Delay,
			MentionAll:      data.MentionAll,
			MentionedJID:    data.MentionedJID,
			FormatJid:       data.FormatJid,
			MediaHandle:     uploaded.Handle,
			ForwardingScore: data.ForwardingScore,
		})

		if err != nil {
			// Check if it's a client disconnection error
			if strings.Contains(err.Error(), "client disconnected") || strings.Contains(err.Error(), "no active session") {
				s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] SendMediaFile failed due to disconnection on attempt %d/%d: %v", instance.Id, attempt, maxRetries, err)
				if attempt < maxRetries {
					waitTime := time.Duration(attempt) * time.Second
					s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Waiting %v before retry", instance.Id, waitTime)
					time.Sleep(waitTime)
					continue
				}
			}
			return nil, err
		}

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendMediaFile successful on attempt %d", instance.Id, attempt)
		return message, nil
	}

	return nil, fmt.Errorf("failed to send media file after %d attempts", maxRetries)
}

func (s *sendService) SendMediaUrl(data *MediaStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	return s.sendMediaUrlWithRetry(data, instance, 3)
}

func (s *sendService) sendMediaUrlWithRetry(data *MediaStruct, instance *instance_model.Instance, maxRetries int) (*MessageSendStruct, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendMediaUrl attempt %d/%d for URL: %s", instance.Id, attempt, maxRetries, data.Url)
		startTime := time.Now()

		client, err := s.ensureClientConnectedWithRetry(instance.Id, 2)
		if err != nil {
			if attempt == maxRetries {
				return nil, err
			}
			continue
		}

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Iniciando download da URL: %s", instance.Id, data.Url)

		resp, err := http.Get(data.Url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Download concluído em %v. Lendo dados...", instance.Id, time.Since(startTime))

		downloadStart := time.Now()
		fileData, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Leitura dos dados concluída em %v. Tamanho: %d bytes", instance.Id, time.Since(downloadStart), len(fileData))

		mime, _ := mimetype.DetectReader(bytes.NewReader(fileData))
		mimeType := mime.String()
		if strings.HasSuffix(strings.ToLower(data.Url), ".mp4") {
			mimeType = "video/mp4"
		}

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Tipo MIME detectado: %s", instance.Id, mimeType)

		var uploadType whatsmeow.MediaType
		var duration int

		processingStart := time.Now()
		switch data.Type {
		case "image":
			if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/webp" {
				errMsg := fmt.Sprintf("Invalid file format: '%s'. Only 'image/jpeg', 'image/png' and 'image/webp' are accepted", mimeType)
				return nil, errors.New(errMsg)
			}
			if mimeType == "image/webp" {
				mimeType = "image/jpeg"
			}
			uploadType = whatsmeow.MediaImage

		case "video", "ptv":
			if mimeType != "video/mp4" {
				errMsg := fmt.Sprintf("Invalid file format: '%s'. Only 'video/mp4' are accepted", mimeType)
				return nil, errors.New(errMsg)
			}
			uploadType = whatsmeow.MediaVideo
		case "audio":
			s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Iniciando conversão de áudio...", instance.Id)
			converterApiUrl := s.config.ApiAudioConverter
			converterApiKey := s.config.ApiAudioConverterKey
			var convertedData []byte
			var err error
			if converterApiUrl == "" {
				s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Usando conversão local...", instance.Id)
				convertedData, duration, err = convertAudioToOpusWithDuration(fileData)
			} else {
				s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Usando API de conversão...", instance.Id)
				convertedData, duration, err = convertAudioWithApi(converterApiUrl, converterApiKey, ConvertAudio{Base64: base64.StdEncoding.EncodeToString(fileData)})
			}
			if err != nil {
				return nil, err
			}
			fileData = convertedData
			mimeType = "audio/ogg; codecs=opus"
			uploadType = whatsmeow.MediaAudio
			s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Conversão de áudio concluída em %v", instance.Id, time.Since(processingStart))
		case "document":
			uploadType = whatsmeow.MediaDocument
		default:
			return nil, errors.New("invalid media type")
		}

		// Detectar se é newsletter para usar upload sem criptografia
		isNewsletter := strings.Contains(data.Number, "@newsletter")

		// Validar se é documento em newsletter (não suportado)
		if isNewsletter && data.Type == "document" {
			return nil, errors.New("documentos não são suportados em canais do WhatsApp. Use imagem, vídeo, áudio ou enquete")
		}

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Iniciando upload para WhatsApp (Newsletter: %v)...", instance.Id, isNewsletter)
		uploadStart := time.Now()

		var uploaded whatsmeow.UploadResponse
		if isNewsletter {
			// Newsletter: upload sem criptografia
			uploaded, err = client.UploadNewsletter(context.Background(), fileData, uploadType)
			s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Newsletter upload - Handle: %s", instance.Id, uploaded.Handle)
		} else {
			// Upload normal com criptografia
			uploaded, err = client.Upload(context.Background(), fileData, uploadType)
		}

		if err != nil {
			return nil, err
		}
		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Upload concluído em %v. Tamanho: %d", instance.Id, time.Since(uploadStart), uploaded.FileLength)

		var media *waE2E.Message
		var mediaType string

		switch data.Type {
		case "image":
			if isNewsletter {
				// Newsletter: sem criptografia (sem MediaKey e FileEncSHA256)
				media = &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
					Caption:    proto.String(data.Caption),
					URL:        &uploaded.URL,
					DirectPath: &uploaded.DirectPath,
					Mimetype:   proto.String(mimeType),
					FileSHA256: uploaded.FileSHA256,
					FileLength: &uploaded.FileLength,
				}}
			} else {
				// Normal: com criptografia
				thumbnail := s.resizeThumbnail(fileData, 100)
				media = &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
					Caption:       proto.String(data.Caption),
					URL:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String(mimeType),
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(fileData))),
					JPEGThumbnail: thumbnail,
				}}
			}
			mediaType = "ImageMessage"
		case "video":
			if isNewsletter {
				media = &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
					Caption:    proto.String(data.Caption),
					URL:        &uploaded.URL,
					DirectPath: &uploaded.DirectPath,
					Mimetype:   proto.String(mimeType),
					FileSHA256: uploaded.FileSHA256,
					FileLength: &uploaded.FileLength,
				}}
			} else {
				thumbnail := s.getVideoThumbnail(fileData)
				media = &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
					Caption:       proto.String(data.Caption),
					URL:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String(mimeType),
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(fileData))),
					JPEGThumbnail: thumbnail,
				}}
			}
			mediaType = "VideoMessage"
		case "ptv":
			if isNewsletter {
				media = &waE2E.Message{PtvMessage: &waE2E.VideoMessage{
					URL:        &uploaded.URL,
					DirectPath: &uploaded.DirectPath,
					Mimetype:   proto.String(mimeType),
					FileSHA256: uploaded.FileSHA256,
					FileLength: &uploaded.FileLength,
				}}
			} else {
				media = &waE2E.Message{PtvMessage: &waE2E.VideoMessage{
					URL:           proto.String(uploaded.URL),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String(mimeType),
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(fileData))),
				}}
			}
			mediaType = "PtvMessage"
		case "audio":
			if isNewsletter {
				media = &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
					URL:              &uploaded.URL,
					PTT:              proto.Bool(true),
					DirectPath:       &uploaded.DirectPath,
					Mimetype:         proto.String(mimeType),
					FileSHA256:       uploaded.FileSHA256,
					FileLength:       &uploaded.FileLength,
					StreamingSidecar: []byte(*proto.String("QpmXDsU7YLagdg==")),
					Waveform:         []byte(*proto.String("OjAnExISDgsKCAkJBwgkHAQEBBEFAwMNAxAcKCgkFzM0QUE4Jh4eKAoKChcLCwkeFgkJCQo3JiQmIiIRPz8/Ow==")),
					Seconds:          proto.Uint32(uint32(duration)),
				}}
			} else {
				media = &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
					URL:              proto.String(uploaded.URL),
					PTT:              proto.Bool(true),
					DirectPath:       proto.String(uploaded.DirectPath),
					MediaKey:         uploaded.MediaKey,
					Mimetype:         proto.String(mimeType),
					FileEncSHA256:    uploaded.FileEncSHA256,
					FileSHA256:       uploaded.FileSHA256,
					FileLength:       proto.Uint64(uploaded.FileLength),
					StreamingSidecar: []byte(*proto.String("QpmXDsU7YLagdg==")),
					Waveform:         []byte(*proto.String("OjAnExISDgsKCAkJBwgkHAQEBBEFAwMNAxAcKCgkFzM0QUE4Jh4eKAoKChcLCwkeFgkJCQo3JiQmIiIRPz8/Ow==")),
					Seconds:          proto.Uint32(uint32(duration)),
				}}
			}
			mediaType = "AudioMessage"
		case "document":
			// For PDF documents, rasterize page 1 into a JPEG preview thumbnail.
			// A missing pdftoppm or a failure yields nil and the document is
			// sent without a preview instead of failing the request.
			var jpegThumb []byte
			if mimeType == "application/pdf" {
				jpegThumb = s.makePDFThumbnail(fileData, 200)
			}
			if isNewsletter {
				media = &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
					URL:           &uploaded.URL,
					FileName:      &data.Filename,
					Caption:       proto.String(data.Caption),
					DirectPath:    &uploaded.DirectPath,
					Mimetype:      proto.String(mimeType),
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    &uploaded.FileLength,
					JPEGThumbnail: jpegThumb,
				}}
			} else {
				media = &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
					URL:           proto.String(uploaded.URL),
					FileName:      &data.Filename,
					Caption:       proto.String(data.Caption),
					DirectPath:    proto.String(uploaded.DirectPath),
					MediaKey:      uploaded.MediaKey,
					Mimetype:      proto.String(mimeType),
					FileEncSHA256: uploaded.FileEncSHA256,
					FileSHA256:    uploaded.FileSHA256,
					FileLength:    proto.Uint64(uint64(len(fileData))),
					JPEGThumbnail: jpegThumb,
				}}
			}

			if media.GetDocumentMessage().GetCaption() != "" {
				media.DocumentWithCaptionMessage = &waE2E.FutureProofMessage{
					Message: &waE2E.Message{
						DocumentMessage: media.DocumentMessage,
					},
				}
				media.DocumentMessage = nil
			}

			mediaType = "DocumentMessage"
		default:
			return nil, errors.New("invalid media type")
		}

		messageStart := time.Now()
		message, err := s.SendMessage(instance, media, mediaType, &SendDataStruct{
			Id:              data.Id,
			Number:          data.Number,
			Quoted:          data.Quoted,
			Delay:           data.Delay,
			MentionAll:      data.MentionAll,
			MentionedJID:    data.MentionedJID,
			FormatJid:       data.FormatJid,
			MediaHandle:     uploaded.Handle,
			ForwardingScore: data.ForwardingScore,
		})

		if err != nil {
			// Check if it's a client disconnection error
			if strings.Contains(err.Error(), "client disconnected") || strings.Contains(err.Error(), "no active session") {
				s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] SendMediaUrl failed due to disconnection on attempt %d/%d: %v", instance.Id, attempt, maxRetries, err)
				if attempt < maxRetries {
					waitTime := time.Duration(attempt) * time.Second
					s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Waiting %v before retry", instance.Id, waitTime)
					time.Sleep(waitTime)
					continue
				}
			}
			return nil, err
		}

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Mensagem enviada em %v", instance.Id, time.Since(messageStart))

		totalTime := time.Since(startTime)
		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendMediaUrl successful on attempt %d, processo completo em %v", instance.Id, attempt, totalTime)

		return message, nil
	}

	return nil, fmt.Errorf("failed to send media url after %d attempts", maxRetries)
}

func (s *sendService) SendPoll(data *PollStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	return s.sendPollWithRetry(data, instance, 3)
}

func (s *sendService) sendPollWithRetry(data *PollStruct, instance *instance_model.Instance, maxRetries int) (*MessageSendStruct, error) {
	for attempt := 1; attempt <= maxRetries; attempt++ {
		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendPoll attempt %d/%d", instance.Id, attempt, maxRetries)

		client, err := s.ensureClientConnectedWithRetry(instance.Id, 2)
		if err != nil {
			if attempt == maxRetries {
				return nil, err
			}
			continue
		}

		msg := client.BuildPollCreation(data.Question, data.Options, data.MaxAnswer)

		message, err := s.SendMessage(instance, msg, "PollCreationMessage", &SendDataStruct{
			Id:           data.Id,
			Number:       data.Number,
			Quoted:       data.Quoted,
			Delay:        data.Delay,
			MentionAll:   data.MentionAll,
			MentionedJID: data.MentionedJID,
			FormatJid:    data.FormatJid,
		})

		if err != nil {
			// Check if it's a client disconnection error
			if strings.Contains(err.Error(), "client disconnected") || strings.Contains(err.Error(), "no active session") {
				s.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] SendPoll failed due to disconnection on attempt %d/%d: %v", instance.Id, attempt, maxRetries, err)
				if attempt < maxRetries {
					waitTime := time.Duration(attempt) * time.Second
					s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Waiting %v before retry", instance.Id, waitTime)
					time.Sleep(waitTime)
					continue
				}
			}
			return nil, err
		}

		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendPoll successful on attempt %d", instance.Id, attempt)
		return message, nil
	}

	return nil, fmt.Errorf("failed to send poll after %d attempts", maxRetries)
}

func convertToWebP(imageData string) ([]byte, error) {
	var img image.Image
	var err error

	resp, err := http.Get(imageData)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image from URL: %v", err)
	}
	defer resp.Body.Close()

	img, _, err = image.Decode(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	var webpBuffer bytes.Buffer
	err = webp.Encode(&webpBuffer, img, &webp.Options{Lossless: false, Quality: 80})
	if err != nil {
		return nil, fmt.Errorf("failed to encode image to WebP: %v", err)
	}

	return webpBuffer.Bytes(), nil
}

func (s *sendService) SendSticker(data *StickerStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	client, err := s.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	var uploaded whatsmeow.UploadResponse
	var filedata []byte

	if strings.HasPrefix(data.Sticker, "http") {
		webpData, err := convertToWebP(data.Sticker)
		if err != nil {
			return nil, fmt.Errorf("failed to convert image to WebP: %v", err)
		}

		filedata = webpData

		uploaded, err = client.Upload(context.Background(), filedata, whatsmeow.MediaImage)
		if err != nil {
			return nil, fmt.Errorf("failed to upload sticker: %v", err)
		}
	} else {
		return nil, fmt.Errorf("invalid sticker URL")
	}

	msg := &waE2E.Message{StickerMessage: &waE2E.StickerMessage{
		URL:           proto.String(uploaded.URL),
		DirectPath:    proto.String(uploaded.DirectPath),
		MediaKey:      uploaded.MediaKey,
		Mimetype:      proto.String(http.DetectContentType(filedata)),
		FileEncSHA256: uploaded.FileEncSHA256,
		FileSHA256:    uploaded.FileSHA256,
		FileLength:    proto.Uint64(uint64(len(filedata))),
	}}

	message, err := s.SendMessage(instance, msg, "StickerMessage", &SendDataStruct{
		Id:           data.Id,
		Number:       data.Number,
		Quoted:       data.Quoted,
		Delay:        data.Delay,
		MentionAll:   data.MentionAll,
		MentionedJID: data.MentionedJID,
		FormatJid:    data.FormatJid,
	})
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (s *sendService) SendLocation(data *LocationStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	_, err := s.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	msg := &waE2E.Message{LocationMessage: &waE2E.LocationMessage{
		DegreesLatitude:  &data.Latitude,
		DegreesLongitude: &data.Longitude,
		Name:             &data.Name,
		Address:          &data.Address,
	}}

	message, err := s.SendMessage(instance, msg, "LocationMessage", &SendDataStruct{
		Id:           data.Id,
		Number:       data.Number,
		Quoted:       data.Quoted,
		Delay:        data.Delay,
		MentionAll:   data.MentionAll,
		MentionedJID: data.MentionedJID,
		FormatJid:    data.FormatJid,
	})
	if err != nil {
		return nil, err
	}

	return message, nil
}

func (s *sendService) SendContact(data *ContactStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	_, err := s.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	VCstring := utils.GenerateVC(utils.VCardStruct{
		FullName:     data.Vcard.FullName,
		Phone:        data.Vcard.Phone,
		Organization: data.Vcard.Organization,
	})

	fmt.Println(VCstring)

	msg := &waE2E.Message{ContactMessage: &waE2E.ContactMessage{
		DisplayName: &data.Vcard.FullName,
		Vcard:       &VCstring,
	}}

	messaged, err := s.SendMessage(instance, msg, "ContactMessage", &SendDataStruct{
		Id:           data.Id,
		Number:       data.Number,
		Quoted:       data.Quoted,
		Delay:        data.Delay,
		MentionAll:   data.MentionAll,
		MentionedJID: data.MentionedJID,
		FormatJid:    data.FormatJid,
	})
	if err != nil {
		return nil, err
	}

	return messaged, nil
}

func mapKeyType(keyType string) string {
	switch keyType {
	case "phone":
		return "PHONE"
	case "email":
		return "EMAIL"
	case "cpf":
		return "CPF"
	case "cnpj":
		return "CNPJ"
	case "random":
		return "EVP"
	default:
		return keyType
	}
}

func (s *sendService) SendButton(data *ButtonStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	client, err := s.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	hasReply := false
	hasPix := false
	hasOtherTypes := false
	replyCount := 0

	for _, v := range data.Buttons {
		switch v.Type {
		case "reply":
			hasReply = true
			replyCount++
		case "pix":
			hasPix = true
		default:
			hasOtherTypes = true
		}
	}

	if hasReply {
		if replyCount > 3 {
			return nil, errors.New("máximo de 3 botões do tipo 'reply' permitidos")
		}
		if hasOtherTypes {
			return nil, errors.New("botões do tipo 'reply' não podem ser misturados com outros tipos")
		}
	}

	if hasPix {
		if len(data.Buttons) > 1 {
			return nil, errors.New("botão do tipo 'pix' não pode ser combinado com outros botões")
		}
	}

	buttons := []*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{}

	for _, v := range data.Buttons {
		var (
			name       string
			paramsData map[string]interface{}
		)

		switch v.Type {
		case "reply":
			name = "quick_reply"
			paramsData = map[string]interface{}{
				"display_text": v.DisplayText,
				"id":           v.Id,
			}
		case "copy":
			name = "cta_copy"
			copyCode := v.CopyCode
			if copyCode == "" {
				copyCode = v.Id
			}
			copyID := v.Id
			if copyID == "" {
				copyID = "copy_" + strconv.FormatInt(time.Now().UnixNano(), 10)
			}
			paramsData = map[string]interface{}{
				"display_text": v.DisplayText,
				"id":           copyID,
				"copy_code":    copyCode,
			}
		case "url":
			name = "cta_url"
			paramsData = map[string]interface{}{
				"display_text": v.DisplayText,
				"url":          v.URL,
				"merchant_url": v.URL,
			}
		case "call":
			name = "cta_call"
			paramsData = map[string]interface{}{
				"display_text": v.DisplayText,
				"phone_number": v.PhoneNumber,
			}
		case "pix":
			randomId := utils.GenerateRandomString(11)
			name = "payment_info"
			amount := map[string]interface{}{"value": 0, "offset": 100}
			paramsData = map[string]interface{}{
				"currency":     v.Currency,
				"total_amount": amount,
				"reference_id": randomId,
				"type":         "physical-goods",
				"order": map[string]interface{}{
					"status":     "pending",
					"subtotal":   amount,
					"order_type": "ORDER",
					"items": []map[string]interface{}{
						{
							"name":        "",
							"amount":      amount,
							"quantity":    0,
							"sale_amount": amount,
						},
					},
				},
				"payment_settings": []map[string]interface{}{
					{
						"type": "pix_static_code",
						"pix_static_code": map[string]interface{}{
							"merchant_name": v.Name,
							"key":           v.Key,
							"key_type":      mapKeyType(v.KeyType),
						},
					},
				},
				"share_payment_status": false,
			}
		default:
			return nil, fmt.Errorf("tipo de botão inválido: %s", v.Type)
		}

		paramsJSONBytes, err := json.Marshal(paramsData)
		if err != nil {
			return nil, err
		}
		paramsJSON := string(paramsJSONBytes)

		buttons = append(buttons, &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
			Name:             proto.String(name),
			ButtonParamsJSON: &paramsJSON,
		})
	}

	templateId := strconv.FormatInt(time.Now().UnixNano()/1000000, 10)
	messageParamsJSON := `{"from":"api","templateId":` + templateId + `}`
	msgSecret := make([]byte, 32)
	_, _ = cryptoRand.Read(msgSecret)

	var msg *waE2E.Message
	msgType := "InteractiveMessage"

	if hasPix {
		paymentMsgParams := `{"native_flow_name":"order_details","version":1}`
		var interactiveBody *waE2E.InteractiveMessage_Body
		if data.Title != "" {
			interactiveBody = &waE2E.InteractiveMessage_Body{Text: &data.Title}
		}

		msg = &waE2E.Message{
			DocumentWithCaptionMessage: &waE2E.FutureProofMessage{
				Message: &waE2E.Message{
					InteractiveMessage: &waE2E.InteractiveMessage{
						Body: interactiveBody,
						InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
							NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
								Buttons:           buttons,
								MessageParamsJSON: &paymentMsgParams,
								MessageVersion:    proto.Int32(1),
							},
						},
					},
				},
			},
			MessageContextInfo: &waE2E.MessageContextInfo{
				MessageSecret: msgSecret,
			},
		}
	} else {
		body := func() string {
			t := "*" + data.Title + "*"
			if data.Description != "" {
				t += "\n\n" + data.Description + "\n"
			}
			return t
		}()

		interactiveMsg := &waE2E.InteractiveMessage{
			Body: &waE2E.InteractiveMessage_Body{
				Text: &body,
			},
			InteractiveMessage: &waE2E.InteractiveMessage_NativeFlowMessage_{
				NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
					Buttons:           buttons,
					MessageParamsJSON: &messageParamsJSON,
					MessageVersion:    proto.Int32(1),
				},
			},
			ContextInfo: &waE2E.ContextInfo{},
		}

		// Footer conditional - only add if not empty (iOS compatibility)
		if data.Footer != "" {
			interactiveMsg.Footer = &waE2E.InteractiveMessage_Footer{
				Text: &data.Footer,
			}
		}

		// Header with title
		if data.Title != "" {
			interactiveMsg.Header = &waE2E.InteractiveMessage_Header{
				Title:              proto.String(data.Title),
				HasMediaAttachment: proto.Bool(false),
			}
		}

		if hasReply {
			var replyButtons []*waE2E.ButtonsMessage_Button
			for _, v := range data.Buttons {
				replyButtons = append(replyButtons, &waE2E.ButtonsMessage_Button{
					ButtonID: proto.String(v.Id),
					ButtonText: &waE2E.ButtonsMessage_Button_ButtonText{
						DisplayText: proto.String(v.DisplayText),
					},
					Type: waE2E.ButtonsMessage_Button_RESPONSE.Enum(),
				})
			}

			buttonsMsg := &waE2E.ButtonsMessage{
				ContentText: proto.String(data.Description),
				FooterText:  proto.String(data.Footer),
				HeaderType:  waE2E.ButtonsMessage_EMPTY.Enum(),
				Buttons:     replyButtons,
			}

			if data.ImageUrl != "" {
				if resp, err := http.Get(data.ImageUrl); err == nil {
					fileData, readErr := io.ReadAll(resp.Body)
					resp.Body.Close()
					if readErr == nil {
						if uploaded, upErr := client.Upload(context.Background(), fileData, whatsmeow.MediaImage); upErr == nil {
							buttonsMsg.HeaderType = waE2E.ButtonsMessage_IMAGE.Enum()
							buttonsMsg.Header = &waE2E.ButtonsMessage_ImageMessage{
								ImageMessage: &waE2E.ImageMessage{
									URL:           proto.String(uploaded.URL),
									DirectPath:    proto.String(uploaded.DirectPath),
									MediaKey:      uploaded.MediaKey,
									Mimetype:      proto.String("image/jpeg"),
									FileEncSHA256: uploaded.FileEncSHA256,
									FileSHA256:    uploaded.FileSHA256,
									FileLength:    proto.Uint64(uint64(len(fileData))),
								},
							}
						}
					}
				}
			} else if data.VideoUrl != "" {
				if resp, err := http.Get(data.VideoUrl); err == nil {
					fileData, readErr := io.ReadAll(resp.Body)
					resp.Body.Close()
					if readErr == nil {
						if uploaded, upErr := client.Upload(context.Background(), fileData, whatsmeow.MediaVideo); upErr == nil {
							buttonsMsg.HeaderType = waE2E.ButtonsMessage_VIDEO.Enum()
							buttonsMsg.Header = &waE2E.ButtonsMessage_VideoMessage{
								VideoMessage: &waE2E.VideoMessage{
									URL:           proto.String(uploaded.URL),
									DirectPath:    proto.String(uploaded.DirectPath),
									MediaKey:      uploaded.MediaKey,
									Mimetype:      proto.String("video/mp4"),
									FileEncSHA256: uploaded.FileEncSHA256,
									FileSHA256:    uploaded.FileSHA256,
									FileLength:    proto.Uint64(uint64(len(fileData))),
								},
							}
						}
					}
				}
			}

			msg = &waE2E.Message{
				DocumentWithCaptionMessage: &waE2E.FutureProofMessage{
					Message: &waE2E.Message{
						ButtonsMessage: buttonsMsg,
					},
				},
				MessageContextInfo: &waE2E.MessageContextInfo{
					MessageSecret: msgSecret,
				},
			}
			msgType = "ButtonsMessage"
		} else {
			msg = &waE2E.Message{
				DocumentWithCaptionMessage: &waE2E.FutureProofMessage{
					Message: &waE2E.Message{
						InteractiveMessage: interactiveMsg,
					},
				},
				MessageContextInfo: &waE2E.MessageContextInfo{
					MessageSecret: msgSecret,
				},
			}
		}
	}

	var additionalNodes *[]waBinary.Node
	if !hasReply || hasOtherTypes || hasPix {
		var nativeFlowName string
		switch {
		case hasPix:
			nativeFlowName = "payment_info"
		default:
			nativeFlowName = "mixed"
		}

		bizNodes := []waBinary.Node{
			{
				Tag: "biz",
				Content: []waBinary.Node{{
					Tag: "interactive",
					Attrs: waBinary.Attrs{
						"type": "native_flow",
						"v":    "1",
					},
					Content: []waBinary.Node{{
						Tag: "native_flow",
						Attrs: waBinary.Attrs{
							"name": nativeFlowName,
						},
					}},
				}},
			},
		}
		if !strings.Contains(data.Number, "@g.us") {
			bizNodes = append(bizNodes, waBinary.Node{
				Tag:   "bot",
				Attrs: waBinary.Attrs{"biz_bot": "1"},
			})
		}
		additionalNodes = &bizNodes
	}

	messaged, err := s.SendMessage(instance, msg, msgType, &SendDataStruct{
		Id:              data.Id,
		Number:          data.Number,
		Quoted:          data.Quoted,
		Delay:           data.Delay,
		MentionAll:      data.MentionAll,
		MentionedJID:    data.MentionedJID,
		FormatJid:       data.FormatJid,
		AdditionalNodes: additionalNodes,
	})
	if err != nil {
		return nil, err
	}

	return messaged, nil
}

func stringPointer(s string) *string {
	return &s
}

func sectionsToString(data *ListStruct) (string, error) {
	type row struct {
		Header      string `json:"header"`
		Title       string `json:"title"`
		Description string `json:"description"`
		ID          string `json:"id"`
	}

	type listSection struct {
		Title          string `json:"title"`
		HighlightLabel string `json:"highlight_label"`
		Rows           []row  `json:"rows"`
	}

	type list struct {
		Title    string        `json:"title"`
		Sections []listSection `json:"sections"`
	}

	sections := []listSection{}

	for _, s := range data.Sections {
		sectionTitle := s.Title
		if sectionTitle == "" {
			sectionTitle = " "
		}
		rows := []row{}

		for _, r := range s.Rows {
			rowTitle := r.Title
			if rowTitle == "" {
				rowTitle = " "
			}
			rowDesc := r.Description
			if rowDesc == "" {
				rowDesc = " "
			}
			rowId := r.RowId
			if rowId == "" {
				rowId = fmt.Sprintf("row_%d", len(rows))
			}
			rows = append(rows, row{
				Header:      rowTitle,
				Title:       rowTitle,
				Description: rowDesc,
				ID:          rowId,
			})
		}

		section := listSection{
			Title:          sectionTitle,
			HighlightLabel: "",
			Rows:           rows,
		}

		sections = append(sections, section)
	}

	buttonText := data.ButtonText
	if buttonText == "" {
		buttonText = "Ver Menu"
	}

	listData := list{
		Title:    buttonText,
		Sections: sections,
	}

	jsonData, err := json.Marshal(listData)
	if err != nil {
		return "", err
	}

	return string(jsonData), nil
}

func (s *sendService) SendList(data *ListStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	// Legacy ListMessage format - works on iOS, Android and Web
	// Matching PAPI Node.js default (non-modern) path exactly

	buttonText := data.ButtonText
	if buttonText == "" {
		buttonText = "Ver Menu"
	}

	// Build sections in legacy ListMessage format
	var sections []*waE2E.ListMessage_Section
	for _, sec := range data.Sections {
		sectionTitle := sec.Title
		if sectionTitle == "" {
			sectionTitle = " "
		}
		var rows []*waE2E.ListMessage_Row
		for i, r := range sec.Rows {
			rowTitle := r.Title
			if rowTitle == "" {
				rowTitle = " "
			}
			rowId := r.RowId
			if rowId == "" {
				rowId = fmt.Sprintf("row_%d_%d", i, len(rows))
			}
			rows = append(rows, &waE2E.ListMessage_Row{
				Title:       proto.String(rowTitle),
				Description: proto.String(r.Description),
				RowID:       proto.String(rowId),
			})
		}
		sections = append(sections, &waE2E.ListMessage_Section{
			Title: proto.String(sectionTitle),
			Rows:  rows,
		})
	}

	listType := waE2E.ListMessage_SINGLE_SELECT
	listMessage := &waE2E.ListMessage{
		Title:       proto.String(data.Title),
		Description: proto.String(data.Description),
		ButtonText:  proto.String(buttonText),
		FooterText:  proto.String(data.FooterText),
		ListType:    &listType,
		Sections:    sections,
	}

	listMsgSecret := make([]byte, 32)
	_, _ = cryptoRand.Read(listMsgSecret)

	msg := &waE2E.Message{
		DocumentWithCaptionMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				ListMessage: listMessage,
			},
		},
		MessageContextInfo: &waE2E.MessageContextInfo{
			MessageSecret: listMsgSecret,
		},
	}

	listBizNodes := []waBinary.Node{
		{
			Tag: "biz",
			Content: []waBinary.Node{{
				Tag: "list",
				Attrs: waBinary.Attrs{
					"v":    "2",
					"type": "product_list",
				},
			}},
		},
	}
	if !strings.Contains(data.Number, "@g.us") {
		listBizNodes = append(listBizNodes, waBinary.Node{
			Tag:   "bot",
			Attrs: waBinary.Attrs{"biz_bot": "1"},
		})
	}

	message, err := s.SendMessage(instance, msg, "ListMessage", &SendDataStruct{
		Id:              data.Id,
		Number:          data.Number,
		Delay:           data.Delay,
		MentionAll:      data.MentionAll,
		MentionedJID:    data.MentionedJID,
		FormatJid:       data.FormatJid,
		Quoted:          data.Quoted,
		AdditionalNodes: &listBizNodes,
	})

	if err != nil {
		s.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error sending list: %v", instance.Id, err)
		return nil, err
	}

	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] List sent to %s", instance.Id, data.Number)
	return message, nil
}

func (s *sendService) SendMessage(instance *instance_model.Instance, msg *waE2E.Message, messageType string, data *SendDataStruct) (*MessageSendStruct, error) {
	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] SendMessage called for number: %s, type: %s", instance.Id, data.Number, messageType)

	recipient, err := s.validateAndCheckUserExists(data.Number, data.FormatJid, &data.Quoted.MessageID, &data.Quoted.MessageID, instance)
	if err != nil {
		s.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error validating message fields or user check: %v", instance.Id, err)
		return nil, err
	}

	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Recipient validated: %s (Server: %s)", instance.Id, recipient.String(), recipient.Server)

	var message string
	if data.Id == "" {
		message = s.clientPointer[instance.Id].GenerateMessageID()
	} else {
		message = data.Id
	}

	if data.Delay > 0 {
		media := ""
		if messageType == "AudioMessage" {
			media = "audio"
		}

		err := s.clientPointer[instance.Id].SendChatPresence(context.Background(), recipient, types.ChatPresence("composing"), types.ChatPresenceMedia(media))
		if err != nil {
			return nil, err
		}

		time.Sleep(time.Duration(data.Delay) * time.Millisecond)

		err = s.clientPointer[instance.Id].SendChatPresence(context.Background(), recipient, types.ChatPresence("paused"), types.ChatPresenceMedia(media))
		if err != nil {
			return nil, err
		}
	}

	// Unwrap message if it's wrapped in ViewOnceMessage for quoting logic
	m := msg
	if m.ViewOnceMessage != nil && m.ViewOnceMessage.Message != nil {
		m = m.ViewOnceMessage.Message
	} else if m.ViewOnceMessageV2 != nil && m.ViewOnceMessageV2.Message != nil {
		m = m.ViewOnceMessageV2.Message
	}

	isMedia := false

	if data.Quoted.MessageID != "" {
		switch messageType {
		case "ExtendedTextMessage":
			if m.ExtendedTextMessage.ContextInfo == nil {
				m.ExtendedTextMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			m.ExtendedTextMessage.ContextInfo.StanzaID = proto.String(data.Quoted.MessageID)
			m.ExtendedTextMessage.ContextInfo.Participant = proto.String(data.Quoted.Participant)
			m.ExtendedTextMessage.ContextInfo.QuotedMessage = &waE2E.Message{Conversation: proto.String("")}
		case "ImageMessage":
			m.ImageMessage.ContextInfo = &waE2E.ContextInfo{
				StanzaID:      proto.String(data.Quoted.MessageID),
				Participant:   proto.String(data.Quoted.Participant),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
			}
			isMedia = true
		case "VideoMessage":
			m.VideoMessage.ContextInfo = &waE2E.ContextInfo{
				StanzaID:      proto.String(data.Quoted.MessageID),
				Participant:   proto.String(data.Quoted.Participant),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
			}
			isMedia = true
		case "PtvMessage":
			m.PtvMessage.ContextInfo = &waE2E.ContextInfo{
				StanzaID:      proto.String(data.Quoted.MessageID),
				Participant:   proto.String(data.Quoted.Participant),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
			}
			isMedia = true
		case "AudioMessage":
			m.AudioMessage.ContextInfo = &waE2E.ContextInfo{
				StanzaID:      proto.String(data.Quoted.MessageID),
				Participant:   proto.String(data.Quoted.Participant),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
			}
			isMedia = true
		case "DocumentMessage":
			if m.DocumentMessage != nil {
				m.DocumentMessage.ContextInfo = &waE2E.ContextInfo{
					StanzaID:      proto.String(data.Quoted.MessageID),
					Participant:   proto.String(data.Quoted.Participant),
					QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
				}
			} else if m.DocumentWithCaptionMessage != nil {
				m.DocumentWithCaptionMessage.Message.DocumentMessage.ContextInfo = &waE2E.ContextInfo{
					StanzaID:      proto.String(data.Quoted.MessageID),
					Participant:   proto.String(data.Quoted.Participant),
					QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
				}
			}
			isMedia = true
		case "PollCreationMessage":
			m.PollCreationMessage.ContextInfo = &waE2E.ContextInfo{
				StanzaID:      proto.String(data.Quoted.MessageID),
				Participant:   proto.String(data.Quoted.Participant),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
			}
		case "StickerMessage":
			m.StickerMessage.ContextInfo = &waE2E.ContextInfo{
				StanzaID:      proto.String(data.Quoted.MessageID),
				Participant:   proto.String(data.Quoted.Participant),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
			}
			isMedia = true
		case "LocationMessage":
			m.LocationMessage.ContextInfo = &waE2E.ContextInfo{
				StanzaID:      proto.String(data.Quoted.MessageID),
				Participant:   proto.String(data.Quoted.Participant),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
			}
		case "ContactMessage":
			m.ContactMessage.ContextInfo = &waE2E.ContextInfo{
				StanzaID:      proto.String(data.Quoted.MessageID),
				Participant:   proto.String(data.Quoted.Participant),
				QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
			}
		case "InteractiveMessage":
			if m.InteractiveMessage != nil {
				m.InteractiveMessage.ContextInfo = &waE2E.ContextInfo{
					StanzaID:      proto.String(data.Quoted.MessageID),
					Participant:   proto.String(data.Quoted.Participant),
					QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
				}
			} else if msg.DocumentWithCaptionMessage != nil &&
				msg.DocumentWithCaptionMessage.Message != nil &&
				msg.DocumentWithCaptionMessage.Message.InteractiveMessage != nil {
				msg.DocumentWithCaptionMessage.Message.InteractiveMessage.ContextInfo = &waE2E.ContextInfo{
					StanzaID:      proto.String(data.Quoted.MessageID),
					Participant:   proto.String(data.Quoted.Participant),
					QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
				}
			}
		case "ListMessage":
			if m.ListMessage != nil {
				m.ListMessage.ContextInfo = &waE2E.ContextInfo{
					StanzaID:      proto.String(data.Quoted.MessageID),
					Participant:   proto.String(data.Quoted.Participant),
					QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
				}
			} else if msg.DocumentWithCaptionMessage != nil &&
				msg.DocumentWithCaptionMessage.Message != nil &&
				msg.DocumentWithCaptionMessage.Message.ListMessage != nil {
				msg.DocumentWithCaptionMessage.Message.ListMessage.ContextInfo = &waE2E.ContextInfo{
					StanzaID:      proto.String(data.Quoted.MessageID),
					Participant:   proto.String(data.Quoted.Participant),
					QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
				}
			}
		case "ButtonsMessage":
			if m.ButtonsMessage != nil {
				m.ButtonsMessage.ContextInfo = &waE2E.ContextInfo{
					StanzaID:      proto.String(data.Quoted.MessageID),
					Participant:   proto.String(data.Quoted.Participant),
					QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
				}
			} else if msg.DocumentWithCaptionMessage != nil &&
				msg.DocumentWithCaptionMessage.Message != nil &&
				msg.DocumentWithCaptionMessage.Message.ButtonsMessage != nil {
				msg.DocumentWithCaptionMessage.Message.ButtonsMessage.ContextInfo = &waE2E.ContextInfo{
					StanzaID:      proto.String(data.Quoted.MessageID),
					Participant:   proto.String(data.Quoted.Participant),
					QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
				}
			}
		default:
			return nil, fmt.Errorf("invalid messageType: %s", messageType)
		}
	} else {
		switch messageType {
		case "ExtendedTextMessage":
			if msg.ExtendedTextMessage.ContextInfo == nil {
				msg.ExtendedTextMessage.ContextInfo = &waE2E.ContextInfo{}
			}
		case "ImageMessage":
			msg.ImageMessage.ContextInfo = &waE2E.ContextInfo{}
			isMedia = true
		case "VideoMessage":
			msg.VideoMessage.ContextInfo = &waE2E.ContextInfo{}
			isMedia = true
		case "PtvMessage":
			msg.PtvMessage.ContextInfo = &waE2E.ContextInfo{}
			isMedia = true
		case "AudioMessage":
			msg.AudioMessage.ContextInfo = &waE2E.ContextInfo{}
			isMedia = true
		case "DocumentMessage":
			if msg.DocumentMessage != nil {
				msg.DocumentMessage.ContextInfo = &waE2E.ContextInfo{}
			} else if msg.DocumentWithCaptionMessage != nil {
				msg.DocumentWithCaptionMessage.Message.DocumentMessage.ContextInfo = &waE2E.ContextInfo{}
			}
			isMedia = true
		case "PollCreationMessage":
			msg.PollCreationMessage.ContextInfo = &waE2E.ContextInfo{}
		case "StickerMessage":
			msg.StickerMessage.ContextInfo = &waE2E.ContextInfo{}
		case "LocationMessage":
			msg.LocationMessage.ContextInfo = &waE2E.ContextInfo{}
		case "ContactMessage":
			msg.ContactMessage.ContextInfo = &waE2E.ContextInfo{}
		case "InteractiveMessage":
			// ContextInfo already set in SendCarousel/SendButton/SendList
		case "ListMessage":
			// ContextInfo already set in SendList
		case "ButtonsMessage":
			// ContextInfo already set in SendButton
		default:
			return nil, fmt.Errorf("invalid messageType: %s", messageType)
		}
	}

	// Apply ForwardingScore to whichever ContextInfo was set above.
	// WhatsApp renders "Encaminhada" when ContextInfo.ForwardingScore > 0.
	if data.ForwardingScore != nil && *data.ForwardingScore > 0 {
		switch messageType {
		case "ExtendedTextMessage":
			if msg.ExtendedTextMessage != nil && msg.ExtendedTextMessage.ContextInfo != nil {
				msg.ExtendedTextMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.ExtendedTextMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "ImageMessage":
			if msg.ImageMessage != nil && msg.ImageMessage.ContextInfo != nil {
				msg.ImageMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.ImageMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "VideoMessage":
			if msg.VideoMessage != nil && msg.VideoMessage.ContextInfo != nil {
				msg.VideoMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.VideoMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "PtvMessage":
			if msg.PtvMessage != nil && msg.PtvMessage.ContextInfo != nil {
				msg.PtvMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.PtvMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "AudioMessage":
			if msg.AudioMessage != nil && msg.AudioMessage.ContextInfo != nil {
				msg.AudioMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.AudioMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "DocumentMessage":
			if msg.DocumentMessage != nil && msg.DocumentMessage.ContextInfo != nil {
				msg.DocumentMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.DocumentMessage.ContextInfo.IsForwarded = proto.Bool(true)
			} else if msg.DocumentWithCaptionMessage != nil && msg.DocumentWithCaptionMessage.Message != nil && msg.DocumentWithCaptionMessage.Message.DocumentMessage != nil && msg.DocumentWithCaptionMessage.Message.DocumentMessage.ContextInfo != nil {
				msg.DocumentWithCaptionMessage.Message.DocumentMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.DocumentWithCaptionMessage.Message.DocumentMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "PollCreationMessage":
			if msg.PollCreationMessage != nil && msg.PollCreationMessage.ContextInfo != nil {
				msg.PollCreationMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.PollCreationMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "StickerMessage":
			if msg.StickerMessage != nil && msg.StickerMessage.ContextInfo != nil {
				msg.StickerMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.StickerMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "LocationMessage":
			if msg.LocationMessage != nil && msg.LocationMessage.ContextInfo != nil {
				msg.LocationMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.LocationMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "ContactMessage":
			if msg.ContactMessage != nil && msg.ContactMessage.ContextInfo != nil {
				msg.ContactMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.ContactMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "InteractiveMessage":
			if msg.InteractiveMessage != nil && msg.InteractiveMessage.ContextInfo != nil {
				msg.InteractiveMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.InteractiveMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		case "ListMessage":
			if msg.ListMessage != nil && msg.ListMessage.ContextInfo != nil {
				msg.ListMessage.ContextInfo.ForwardingScore = data.ForwardingScore
				msg.ListMessage.ContextInfo.IsForwarded = proto.Bool(true)
			}
		}
	}

	isGroup := strings.Contains(data.Number, "@g.us")
	isNewsletter := strings.Contains(data.Number, "@newsletter")

	// Only try to get participants for actual groups, not newsletters
	if isGroup && !isNewsletter {
		if data.MentionAll {
			groupInfo, err := s.clientPointer[instance.Id].GetGroupInfo(context.Background(), recipient)
			if err != nil {
				return nil, err
			}

			var mentionedJIDs []string
			for _, participant := range groupInfo.Participants {
				mentionedJIDs = append(mentionedJIDs, participant.JID.String())
			}

			switch messageType {
			case "ExtendedTextMessage":
				if msg.ExtendedTextMessage.ContextInfo == nil {
					msg.ExtendedTextMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.ExtendedTextMessage.ContextInfo.MentionedJID = mentionedJIDs
			case "ImageMessage":
				if msg.ImageMessage.ContextInfo == nil {
					msg.ImageMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.ImageMessage.ContextInfo.MentionedJID = mentionedJIDs
			case "VideoMessage":
				if msg.VideoMessage.ContextInfo == nil {
					msg.VideoMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.VideoMessage.ContextInfo.MentionedJID = mentionedJIDs
			case "PtvMessage":
				if msg.PtvMessage.ContextInfo == nil {
					msg.PtvMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.PtvMessage.ContextInfo.MentionedJID = mentionedJIDs
			case "AudioMessage":
				if msg.AudioMessage.ContextInfo == nil {
					msg.AudioMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.AudioMessage.ContextInfo.MentionedJID = mentionedJIDs
			case "DocumentMessage":
				if msg.DocumentMessage.ContextInfo == nil {
					msg.DocumentMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.DocumentMessage.ContextInfo.MentionedJID = mentionedJIDs
			case "PollCreationMessage":
				if msg.PollCreationMessage.ContextInfo == nil {
					msg.PollCreationMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.PollCreationMessage.ContextInfo.MentionedJID = mentionedJIDs
			case "StickerMessage":
				if msg.StickerMessage.ContextInfo == nil {
					msg.StickerMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.StickerMessage.ContextInfo.MentionedJID = mentionedJIDs
			case "LocationMessage":
				if msg.LocationMessage.ContextInfo == nil {
					msg.LocationMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.LocationMessage.ContextInfo.MentionedJID = mentionedJIDs
			case "ContactMessage":
				if msg.ContactMessage.ContextInfo == nil {
					msg.ContactMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.ContactMessage.ContextInfo.MentionedJID = mentionedJIDs
			}

		}

		if len(data.MentionedJID) > 0 {
			switch messageType {
			case "ExtendedTextMessage":
				if msg.ExtendedTextMessage.ContextInfo == nil {
					msg.ExtendedTextMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.ExtendedTextMessage.ContextInfo.MentionedJID = data.MentionedJID
			case "ImageMessage":
				if msg.ImageMessage.ContextInfo == nil {
					msg.ImageMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.ImageMessage.ContextInfo.MentionedJID = data.MentionedJID
			case "VideoMessage":
				if msg.VideoMessage.ContextInfo == nil {
					msg.VideoMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.VideoMessage.ContextInfo.MentionedJID = data.MentionedJID
			case "PtvMessage":
				if msg.PtvMessage.ContextInfo == nil {
					msg.PtvMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.PtvMessage.ContextInfo.MentionedJID = data.MentionedJID
			case "AudioMessage":
				if msg.AudioMessage.ContextInfo == nil {
					msg.AudioMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.AudioMessage.ContextInfo.MentionedJID = data.MentionedJID
			case "DocumentMessage":
				if msg.DocumentMessage.ContextInfo == nil {
					msg.DocumentMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.DocumentMessage.ContextInfo.MentionedJID = data.MentionedJID
			case "PollCreationMessage":
				if msg.PollCreationMessage.ContextInfo == nil {
					msg.PollCreationMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.PollCreationMessage.ContextInfo.MentionedJID = data.MentionedJID
			case "StickerMessage":
				if msg.StickerMessage.ContextInfo == nil {
					msg.StickerMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.StickerMessage.ContextInfo.MentionedJID = data.MentionedJID
			case "LocationMessage":
				if msg.LocationMessage.ContextInfo == nil {
					msg.LocationMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.LocationMessage.ContextInfo.MentionedJID = data.MentionedJID
			case "ContactMessage":
				if msg.ContactMessage.ContextInfo == nil {
					msg.ContactMessage.ContextInfo = &waE2E.ContextInfo{}
				}
				msg.ContactMessage.ContextInfo.MentionedJID = data.MentionedJID
			}
		}
	}

	recipient.User = strings.ReplaceAll(recipient.User, "+", "")

	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Sending message to %s with ID %s", instance.Id, recipient.String(), message)

	// Preparar extra parameters para o envio
	sendExtra := whatsmeow.SendRequestExtra{ID: message}

	// Para newsletters/canais, adicionar o MediaHandle se houver mídia
	if recipient.Server == "newsletter" && data.MediaHandle != "" {
		sendExtra.MediaHandle = data.MediaHandle
		s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Newsletter detected, using MediaHandle: %s", instance.Id, data.MediaHandle)
	}

	if data.AdditionalNodes != nil {
		sendExtra.AdditionalNodes = data.AdditionalNodes
	}

	response, err := s.clientPointer[instance.Id].SendMessage(context.Background(), recipient, msg, sendExtra)
	if err != nil {
		s.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error sending message: %v", instance.Id, err)
		return nil, err
	}

	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Message sent successfully! ServerID: %d", instance.Id, response.ServerID)

	messageInfo := types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     recipient,
			Sender:   *s.clientPointer[instance.Id].Store.ID,
			IsFromMe: true,
			IsGroup:  isGroup,
		},
		ID:        message,
		Timestamp: time.Now(),
		ServerID:  response.ServerID,
		Type:      messageType,
	}

	messageSent := &MessageSendStruct{
		Info:    messageInfo,
		Message: msg,
		MessageContextInfo: &waE2E.ContextInfo{
			StanzaID:      proto.String(data.Quoted.MessageID),
			Participant:   proto.String(data.Quoted.Participant),
			QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
		},
	}

	postMap := make(map[string]interface{})
	postMap["event"] = "SendMessage"

	// Convertendo o MessageSendStruct para map antes de atribuir
	messageData := make(map[string]interface{})
	messageData["Info"] = messageSent.Info

	// Convertendo a mensagem para map usando json marshal/unmarshal
	msgBytes, err := json.Marshal(messageSent.Message)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal message: %v", err)
	}

	var msgMap map[string]interface{}
	if err := json.Unmarshal(msgBytes, &msgMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %v", err)
	}

	messageData["Message"] = msgMap
	messageData["MessageContextInfo"] = messageSent.MessageContextInfo

	postMap["data"] = messageData

	if isMedia && s.config.WebhookFiles {
		var data []byte
		var err error

		img := msg.GetImageMessage()
		audio := msg.GetAudioMessage()
		document := msg.GetDocumentMessage()
		video := msg.GetVideoMessage()
		sticker := msg.GetStickerMessage()

		if img != nil {
			data, err = s.clientPointer[instance.Id].Download(context.Background(), img)
		} else if audio != nil {
			data, err = s.clientPointer[instance.Id].Download(context.Background(), audio)
		} else if document != nil {
			data, err = s.clientPointer[instance.Id].Download(context.Background(), document)
		} else if video != nil {
			data, err = s.clientPointer[instance.Id].Download(context.Background(), video)
		} else if sticker != nil {
			data, err = s.clientPointer[instance.Id].Download(context.Background(), sticker)

			webpReader := bytes.NewReader(data)
			img, err := webp.Decode(webpReader)
			if err == nil {
				var pngBuffer bytes.Buffer
				err = png.Encode(&pngBuffer, img)
				if err == nil {
					data = pngBuffer.Bytes()
				}
			}
		}

		if err == nil {
			// Acessando o Message do map já convertido
			messageMap := msgMap
			if messageMap == nil {
				messageMap = make(map[string]interface{})
			}

			encodeData := base64.StdEncoding.EncodeToString(data)
			messageMap["base64"] = encodeData

			messageData["Message"] = messageMap
		}
	}

	postMap["instanceToken"] = instance.Token
	postMap["instanceId"] = instance.Id
	postMap["instanceName"] = instance.Name

	var queueName string

	if _, ok := postMap["event"]; ok {
		queueName = strings.ToLower(fmt.Sprintf("%s.%s", instance.Id, postMap["event"]))
	}

	values, err := json.Marshal(postMap)
	if err != nil {
		s.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Failed to marshal JSON for queue", instance.Id)
		return nil, err
	}

	go s.whatsmeowService.CallWebhook(instance, queueName, values)

	if s.config.AmqpGlobalEnabled || s.config.NatsGlobalEnabled {
		go s.whatsmeowService.SendToGlobalQueues(postMap["event"].(string), values, instance.Id)
	}

	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Message sent to %s", instance.Id, data.Number)
	return messageSent, nil
}

func (s *sendService) SendCarousel(data *CarouselStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	client, err := s.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	formatJid := true
	if data.FormatJid != nil {
		formatJid = *data.FormatJid
	}

	var recipient types.JID
	var ok bool
	recipient, ok = utils.ParseJID(data.Number)
	if !ok && formatJid {
		s.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error validating message fields", instance.Id)
		return nil, errors.New("invalid phone number")
	} else if !ok && !formatJid {
		recipient = types.JID{
			User:   data.Number,
			Server: types.DefaultUserServer,
		}
	}

	// Build carousel cards
	cards := make([]*waE2E.InteractiveMessage, len(data.Cards))
	messageVersion := int32(1)

	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Building carousel for %s with %d cards", instance.Id, recipient.String(), len(data.Cards))

	for i, card := range data.Cards {
		// Each card MUST have both header and body for carousel to work
		interactiveCard := &waE2E.InteractiveMessage{
			Body: &waE2E.InteractiveMessage_Body{
				Text: proto.String(card.Body.Text),
			},
			Header: &waE2E.InteractiveMessage_Header{
				Title:              proto.String(card.Header.Title),
				Subtitle:           proto.String(card.Header.Subtitle),
				HasMediaAttachment: proto.Bool(false),
			},
		}

		// Add media to header if URL provided
		if card.Header.ImageUrl != "" || card.Header.VideoUrl != "" {
			header := interactiveCard.Header

			if card.Header.ImageUrl != "" {
				// Download image
				resp, err := http.Get(card.Header.ImageUrl)
				if err == nil {
					defer resp.Body.Close()
					fileData, err := io.ReadAll(resp.Body)
					if err == nil {
						uploaded, err := client.Upload(context.Background(), fileData, whatsmeow.MediaImage)
						if err == nil {
							header.HasMediaAttachment = proto.Bool(true)
							thumbnail := s.resizeThumbnail(fileData, 600)
							header.Media = &waE2E.InteractiveMessage_Header_ImageMessage{
								ImageMessage: &waE2E.ImageMessage{
									URL:           proto.String(uploaded.URL),
									DirectPath:    proto.String(uploaded.DirectPath),
									MediaKey:      uploaded.MediaKey,
									Mimetype:      proto.String("image/jpeg"),
									FileEncSHA256: uploaded.FileEncSHA256,
									FileSHA256:    uploaded.FileSHA256,
									FileLength:    proto.Uint64(uint64(len(fileData))),
									JPEGThumbnail: thumbnail,
								},
							}

						}
					}
				}
			} else if card.Header.VideoUrl != "" {
				// Download and upload video
				resp, err := http.Get(card.Header.VideoUrl)
				if err == nil {
					defer resp.Body.Close()
					fileData, err := io.ReadAll(resp.Body)
					if err == nil {
						uploaded, err := client.Upload(context.Background(), fileData, whatsmeow.MediaVideo)
						if err == nil {
							header.HasMediaAttachment = proto.Bool(true)
							header.Media = &waE2E.InteractiveMessage_Header_VideoMessage{
								VideoMessage: &waE2E.VideoMessage{
									URL:           proto.String(uploaded.URL),
									DirectPath:    proto.String(uploaded.DirectPath),
									MediaKey:      uploaded.MediaKey,
									Mimetype:      proto.String("video/mp4"),
									FileEncSHA256: uploaded.FileEncSHA256,
									FileSHA256:    uploaded.FileSHA256,
									FileLength:    proto.Uint64(uint64(len(fileData))),
								},
							}
						}
					}
				}
			}
		}

		// Add footer if exists
		if card.Footer != "" {
			interactiveCard.Footer = &waE2E.InteractiveMessage_Footer{
				Text: proto.String(card.Footer),
			}
		}

		// Add buttons if exist
		if len(card.Buttons) > 0 {
			buttons := make([]*waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton, len(card.Buttons))
			for j, btn := range card.Buttons {
				buttonType := strings.ToUpper(btn.Type)
				if buttonType == "" {
					buttonType = "REPLY" // Default type
				}

				var buttonName string
				var buttonParams string

				switch buttonType {
				case "URL":
					// URL button - opens a link
					buttonName = "cta_url"
					buttonParams = fmt.Sprintf(`{"display_text":"%s","url":"%s"}`, btn.DisplayText, btn.Id)
				case "CALL":
					// Call button - initiates a phone call
					buttonName = "cta_call"
					buttonParams = fmt.Sprintf(`{"display_text":"%s","phone_number":"%s"}`, btn.DisplayText, btn.Id)
				case "COPY":
					// Copy button - copies text to clipboard
					buttonName = "cta_copy"
					buttonParams = fmt.Sprintf(`{"display_text":"%s","copy_code":"%s"}`, btn.DisplayText, btn.CopyCode)
				case "REPLY":
					fallthrough
				default:
					// Quick reply button (default)
					buttonName = "quick_reply"
					buttonParams = fmt.Sprintf(`{"display_text":"%s","id":"%s"}`, btn.DisplayText, btn.Id)
				}

				buttons[j] = &waE2E.InteractiveMessage_NativeFlowMessage_NativeFlowButton{
					Name:             proto.String(buttonName),
					ButtonParamsJSON: proto.String(buttonParams),
				}
			}

			// Cards in carousel: do NOT set MessageParamsJSON or MessageVersion
			// (matching PAPI Node.js behavior for iOS compatibility)
			interactiveCard.InteractiveMessage = &waE2E.InteractiveMessage_NativeFlowMessage_{
				NativeFlowMessage: &waE2E.InteractiveMessage_NativeFlowMessage{
					Buttons: buttons,
				},
			}
		}

		cards[i] = interactiveCard
	}

	// Build carousel message (do NOT set CarouselCardType - matching PAPI Node.js for iOS)
	interactiveMsg := &waE2E.InteractiveMessage{
		InteractiveMessage: &waE2E.InteractiveMessage_CarouselMessage_{
			CarouselMessage: &waE2E.InteractiveMessage_CarouselMessage{
				Cards:          cards,
				MessageVersion: &messageVersion,
			},
		},
	}

	// Add body if provided (main message above carousel)
	if data.Body != "" {
		interactiveMsg.Body = &waE2E.InteractiveMessage_Body{
			Text: proto.String(data.Body),
		}
	}

	// Add footer if provided (text below carousel)
	if data.Footer != "" {
		interactiveMsg.Footer = &waE2E.InteractiveMessage_Footer{
			Text: proto.String(data.Footer),
		}
	}

	// ContextInfo is REQUIRED for iOS compatibility
	// Even if empty, iOS requires this field to display carousel
	contextInfo := &waE2E.ContextInfo{}

	// Add quoted message if exists
	if data.Quoted.MessageID != "" {
		contextInfo.StanzaID = proto.String(data.Quoted.MessageID)
		if data.Quoted.Participant != "" {
			participantJID, ok := utils.ParseJID(data.Quoted.Participant)
			if ok {
				contextInfo.Participant = proto.String(participantJID.String())
			}
		}
	}

	// Always set ContextInfo (required for iOS)
	interactiveMsg.ContextInfo = contextInfo

	// Build final message with MessageContextInfo for proper notification delivery
	msg := &waE2E.Message{
		ViewOnceMessage: &waE2E.FutureProofMessage{
			Message: &waE2E.Message{
				InteractiveMessage: interactiveMsg,
			},
		},
		MessageContextInfo: &waE2E.MessageContextInfo{
			DeviceListMetadata: &waE2E.DeviceListMetadata{},
		},
	}

	message, err := s.SendMessage(instance, msg, "InteractiveMessage", &SendDataStruct{
		Id:     data.Id,
		Number: data.Number,
		Delay:  data.Delay,
		Quoted: data.Quoted,
	})

	if err != nil {
		s.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error sending carousel: %v", instance.Id, err)
		return nil, err
	}

	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Carousel sent to %s with %d cards", instance.Id, data.Number, len(data.Cards))
	return message, nil
}

func (s *sendService) SendStatusText(data *StatusTextStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	client, err := s.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	if data.Text == "" {
		return nil, errors.New("text is required")
	}

	msg := &waE2E.Message{
		ExtendedTextMessage: &waE2E.ExtendedTextMessage{
			Text: &data.Text,
		},
	}

	messageID := data.Id
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	recipient := types.NewJID("status", "broadcast")

	response, err := client.SendMessage(context.Background(), recipient, msg, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return nil, err
	}

	messageInfo := types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     recipient,
			Sender:   *client.Store.ID,
			IsFromMe: true,
			IsGroup:  false,
		},
		ID:        messageID,
		Timestamp: time.Now(),
		ServerID:  response.ServerID,
		Type:      "StatusTextMessage",
	}

	messageSent := &MessageSendStruct{
		Info:    messageInfo,
		Message: msg,
		MessageContextInfo: &waE2E.ContextInfo{
			StanzaID:      proto.String(""),
			Participant:   proto.String(""),
			QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
		},
	}

	s.sendStatusWebhook(messageSent, instance, "text")
	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Status text sent successfully", instance.Id)
	return messageSent, nil
}

func (s *sendService) SendStatusMediaUrl(data *StatusMediaStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	client, err := s.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	if data.Url == "" {
		return nil, errors.New("url is required")
	}
	if data.Type != "image" && data.Type != "video" {
		return nil, errors.New("type must be 'image' or 'video'")
	}

	req, err := http.NewRequest("GET", data.Url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Evolution-GO/1.0")

	httpClient := &http.Client{}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download file from URL: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("failed to download file: HTTP status %d", resp.StatusCode)
	}

	fileData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.sendStatusMedia(client, data, fileData, instance)
}

func (s *sendService) SendStatusMediaFile(data *StatusMediaStruct, fileData []byte, instance *instance_model.Instance) (*MessageSendStruct, error) {
	client, err := s.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	if data.Type != "image" && data.Type != "video" {
		return nil, errors.New("type must be 'image' or 'video'")
	}

	return s.sendStatusMedia(client, data, fileData, instance)
}

func (s *sendService) sendStatusMedia(client *whatsmeow.Client, data *StatusMediaStruct, fileData []byte, instance *instance_model.Instance) (*MessageSendStruct, error) {
	mime, _ := mimetype.DetectReader(bytes.NewReader(fileData))
	mimeType := mime.String()

	var uploadType whatsmeow.MediaType
	switch data.Type {
	case "image":
		if mimeType != "image/jpeg" && mimeType != "image/png" && mimeType != "image/webp" {
			return nil, fmt.Errorf("invalid file format: '%s'. Only 'image/jpeg', 'image/png' and 'image/webp' are accepted", mimeType)
		}
		if mimeType == "image/webp" {
			mimeType = "image/jpeg"
		}
		uploadType = whatsmeow.MediaImage
	case "video":
		if mimeType != "video/mp4" {
			return nil, fmt.Errorf("invalid file format: '%s'. Only 'video/mp4' is accepted", mimeType)
		}
		uploadType = whatsmeow.MediaVideo
	default:
		return nil, errors.New("invalid media type")
	}

	uploaded, err := client.Upload(context.Background(), fileData, uploadType)
	if err != nil {
		return nil, err
	}

	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Status media uploaded, size: %d", instance.Id, uploaded.FileLength)

	var media *waE2E.Message
	var mediaType string

	switch data.Type {
	case "image":
		media = &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(data.Caption),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(fileData))),
		}}
		mediaType = "ImageMessage"
	case "video":
		media = &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
			Caption:       proto.String(data.Caption),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			Mimetype:      proto.String(mimeType),
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(fileData))),
		}}
		mediaType = "VideoMessage"
	}

	messageID := data.Id
	if messageID == "" {
		messageID = client.GenerateMessageID()
	}

	recipient := types.NewJID("status", "broadcast")

	response, err := client.SendMessage(context.Background(), recipient, media, whatsmeow.SendRequestExtra{ID: messageID})
	if err != nil {
		return nil, err
	}

	messageInfo := types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     recipient,
			Sender:   *client.Store.ID,
			IsFromMe: true,
			IsGroup:  false,
		},
		ID:        messageID,
		Timestamp: time.Now(),
		ServerID:  response.ServerID,
		Type:      mediaType,
	}

	messageSent := &MessageSendStruct{
		Info:    messageInfo,
		Message: media,
		MessageContextInfo: &waE2E.ContextInfo{
			StanzaID:      proto.String(""),
			Participant:   proto.String(""),
			QuotedMessage: &waE2E.Message{Conversation: proto.String("")},
		},
	}

	s.sendStatusWebhook(messageSent, instance, "media")
	return messageSent, nil
}

func (s *sendService) sendStatusWebhook(messageSent *MessageSendStruct, instance *instance_model.Instance, messageType string) {
	postMap := make(map[string]interface{})
	postMap["event"] = "SendStatus"
	messageData := make(map[string]interface{})
	messageData["Info"] = messageSent.Info
	msgBytes, err := json.Marshal(messageSent.Message)
	if err != nil {
		s.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Failed to marshal status message: %v", instance.Id, err)
		return
	}
	var msgMap map[string]interface{}
	if err := json.Unmarshal(msgBytes, &msgMap); err != nil {
		s.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Failed to unmarshal status message: %v", instance.Id, err)
		return
	}
	messageData["Message"] = msgMap
	messageData["MessageContextInfo"] = messageSent.MessageContextInfo
	postMap["data"] = messageData
	postMap["instanceToken"] = instance.Token
	postMap["instanceId"] = instance.Id
	postMap["instanceName"] = instance.Name

	values, err := json.Marshal(postMap)
	if err != nil {
		s.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Failed to marshal webhook payload: %v", instance.Id, err)
		return
	}
	go s.whatsmeowService.CallWebhook(instance, "sendstatus", values)
	if s.config.AmqpGlobalEnabled || s.config.NatsGlobalEnabled {
		go s.whatsmeowService.SendToGlobalQueues("SendStatus", values, instance.Id)
	}
	s.loggerWrapper.GetLogger(instance.Id).LogInfo("[%s] Status %s sent successfully", instance.Id, messageType)
}

func NewSendService(
	clientPointer map[string]*whatsmeow.Client,
	whatsmeowService whatsmeow_service.WhatsmeowService,
	config *config.Config,
	loggerWrapper *logger_wrapper.LoggerManager,
) SendService {
	return &sendService{
		clientPointer:    clientPointer,
		whatsmeowService: whatsmeowService,
		config:           config,
		loggerWrapper:    loggerWrapper,
	}
}
