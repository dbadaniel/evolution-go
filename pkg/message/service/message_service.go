package message_service

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	instance_model "github.com/EvolutionAPI/evolution-go/pkg/instance/model"
	logger_wrapper "github.com/EvolutionAPI/evolution-go/pkg/logger"
	message_model "github.com/EvolutionAPI/evolution-go/pkg/message/model"
	message_repository "github.com/EvolutionAPI/evolution-go/pkg/message/repository"
	"github.com/EvolutionAPI/evolution-go/pkg/utils"
	whatsmeow_service "github.com/EvolutionAPI/evolution-go/pkg/whatsmeow/service"
	"github.com/vincent-petithory/dataurl"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waCommon"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/types"
	"google.golang.org/protobuf/proto"
)

type MessageService interface {
	React(data *ReactStruct, instance *instance_model.Instance) (*MessageSendStruct, error)
	ChatPresence(data *ChatPresenceStruct, instance *instance_model.Instance) (string, error)
	MarkRead(data *MarkReadStruct, instance *instance_model.Instance) (string, error)
	MarkPlayed(data *MarkPlayedStruct, instance *instance_model.Instance) (string, error)
	DownloadMedia(data *DownloadMediaStruct, instance *instance_model.Instance, request *http.Request) (*dataurl.DataURL, string, error)
	GetMessageStatus(data *MessageStatusStruct, instance *instance_model.Instance) (*message_model.Message, string, error)
	DeleteMessageEveryone(data *MessageStruct, instance *instance_model.Instance) (string, string, error)
	EditMessage(data *EditMessageStruct, instance *instance_model.Instance) (string, string, error)
	PinMessage(data *PinMessageStruct, instance *instance_model.Instance) (string, string, error)
	UnpinMessage(data *PinMessageStruct, instance *instance_model.Instance) (string, string, error)
}

type messageService struct {
	clientPointer     map[string]*whatsmeow.Client
	messageRepository message_repository.MessageRepository
	whatsmeowService  whatsmeow_service.WhatsmeowService
	loggerWrapper     *logger_wrapper.LoggerManager
}

type ReactStruct struct {
	Number      string `json:"number"`
	Reaction    string `json:"reaction"`
	Id          string `json:"id"`
	FromMe      bool   `json:"fromMe"`
	Participant string `json:"participant,omitempty"`
}

type ChatPresenceStruct struct {
	Number  string `json:"number"`
	State   string `json:"state"`
	IsAudio bool   `json:"isAudio"`
	Delay   int    `json:"delay"`
}

type MarkReadStruct struct {
	Id     []string `json:"id"`
	Number string   `json:"number"`
}

type MarkPlayedStruct struct {
	Id     []string `json:"id"`
	Number string   `json:"number"`
}

type DownloadMediaStruct struct {
	Message *waE2E.Message `json:"message"`
}

type MessageStatusStruct struct {
	Id string `json:"id"`
}

type MessageStruct struct {
	Chat      string `json:"chat"`
	MessageID string `json:"messageId"`
}

type PinMessageStruct struct {
	Chat        string `json:"chat"`
	Number      string `json:"number"`
	MessageID   string `json:"messageId"`
	FromMe      bool   `json:"fromMe"`
	Participant string `json:"participant,omitempty"`
}

type EditMessageStruct struct {
	Chat      string `json:"chat"`
	Message   string `json:"message"`
	MessageID string `json:"messageId"`
}

type MessageSendStruct struct {
	Info               types.MessageInfo
	Message            *waE2E.Message
	MessageContextInfo *waE2E.ContextInfo
}

func (m *messageService) ensureClientConnected(instanceId string) (*whatsmeow.Client, error) {
	client := m.clientPointer[instanceId]
	m.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Checking client connection status - Client exists: %v", instanceId, client != nil)

	if client == nil {
		m.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] No client found, attempting to start new instance", instanceId)
		err := m.whatsmeowService.StartInstance(instanceId)
		if err != nil {
			m.loggerWrapper.GetLogger(instanceId).LogError("[%s] Failed to start instance: %v", instanceId, err)
			return nil, errors.New("no active session found")
		}

		m.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Instance started, waiting 2 seconds...", instanceId)
		time.Sleep(2 * time.Second)

		client = m.clientPointer[instanceId]
		m.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Checking new client - Exists: %v, Connected: %v",
			instanceId,
			client != nil,
			client != nil && client.IsConnected())

		if client == nil || !client.IsConnected() {
			m.loggerWrapper.GetLogger(instanceId).LogError("[%s] New client validation failed - Exists: %v, Connected: %v",
				instanceId,
				client != nil,
				client != nil && client.IsConnected())
			return nil, errors.New("no active session found")
		}
	} else if !client.IsConnected() {
		m.loggerWrapper.GetLogger(instanceId).LogError("[%s] Existing client is disconnected - Connected status: %v",
			instanceId,
			client.IsConnected())
		return nil, errors.New("client disconnected")
	}

	m.loggerWrapper.GetLogger(instanceId).LogInfo("[%s] Client successfully validated - Connected: %v", instanceId, client.IsConnected())
	return client, nil
}

func (m *messageService) React(data *ReactStruct, instance *instance_model.Instance) (*MessageSendStruct, error) {
	client, err := m.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, err
	}

	msgId := ""

	recipient, ok := utils.ParseJID(data.Number)
	if !ok {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error validating message fields", instance.Id)
		return nil, errors.New("invalid phone number")
	}
	recipient = utils.CanonicalJID(recipient)

	if data.Id == "" {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Missing Id in Payload", instance.Id)
		return nil, errors.New("missing id in payload")
	} else {
		msgId = data.Id
	}

	fromMe := data.FromMe
	reaction := data.Reaction
	if reaction == "remove" {
		reaction = ""
	}

	// Create MessageKey
	messageKey := &waCommon.MessageKey{
		RemoteJID: proto.String(recipient.String()),
		FromMe:    proto.Bool(fromMe),
		ID:        proto.String(msgId),
	}

	// Add participant if provided (for group messages)
	if data.Participant != "" {
		participantJID, ok := utils.ParseJID(data.Participant)
		if ok {
			messageKey.Participant = proto.String(utils.CanonicalJID(participantJID).String())
		}
	}

	msg := &waE2E.Message{
		ReactionMessage: &waE2E.ReactionMessage{
			Key:  messageKey,
			Text: proto.String(reaction),
			// GroupingKey:       proto.String(reaction),
			SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
		},
	}

	response, err := client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		return nil, err
	}

	isGroup := strings.Contains(data.Number, "@g.us")
	messageType := "ReactionMessage"

	messageInfo := types.MessageInfo{
		MessageSource: types.MessageSource{
			Chat:     recipient,
			Sender:   *client.Store.ID,
			IsFromMe: true,
			IsGroup:  isGroup,
		},
		ID:        response.ID,
		Timestamp: time.Now(),
		ServerID:  response.ServerID,
		Type:      messageType,
	}

	messageSent := &MessageSendStruct{
		Info:    messageInfo,
		Message: msg,
	}

	return messageSent, nil
}

func (m *messageService) ChatPresence(data *ChatPresenceStruct, instance *instance_model.Instance) (string, error) {
	client, err := m.ensureClientConnected(instance.Id)
	if err != nil {
		return "", err
	}

	var ts time.Time

	recipient, ok := utils.ParseJID(data.Number)
	if !ok {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error validating message fields", instance.Id)
		return "", errors.New("invalid phone number")
	}
	recipient = utils.CanonicalJID(recipient)

	media := ""

	if data.IsAudio {
		media = "audio"
	}

	if presErr := client.SendPresence(context.Background(), types.PresenceAvailable); presErr != nil {
		m.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] SendPresence(available) before chatstate failed (non-fatal): %v", instance.Id, presErr)
	}

	state := types.ChatPresence(data.State)
	mediaType := types.ChatPresenceMedia(media)
	err = client.SendChatPresence(context.Background(), recipient, state, mediaType)
	if err != nil {
		return "", err
	}

	if data.Delay > 0 && state == types.ChatPresenceComposing {
		const keepAliveInterval = 5 * time.Second
		const maxDelay = 60 * time.Second
		remaining := time.Duration(data.Delay) * time.Millisecond
		if remaining > maxDelay {
			remaining = maxDelay
		}
		for remaining > 0 {
			sleep := keepAliveInterval
			if remaining < sleep {
				sleep = remaining
			}
			time.Sleep(sleep)
			remaining -= sleep
			if remaining > 0 {
				if refreshErr := client.SendChatPresence(context.Background(), recipient, state, mediaType); refreshErr != nil {
					m.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] Refresh chatstate failed (non-fatal): %v", instance.Id, refreshErr)
				}
			}
		}
		if pausedErr := client.SendChatPresence(context.Background(), recipient, types.ChatPresencePaused, mediaType); pausedErr != nil {
			m.loggerWrapper.GetLogger(instance.Id).LogWarn("[%s] SendChatPresence(paused) failed (non-fatal): %v", instance.Id, pausedErr)
		}
	}

	m.loggerWrapper.GetLogger(instance.Id).LogInfo("Presence (%s) sent to %s", data.State, data.Number)

	return ts.String(), nil
}

func (m *messageService) MarkRead(data *MarkReadStruct, instance *instance_model.Instance) (string, error) {
	client, err := m.ensureClientConnected(instance.Id)
	if err != nil {
		return "", err
	}

	var ts time.Time

	jid, ok := utils.ParseJID(data.Number)
	if !ok {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error validating message fields", instance.Id)
		return "", errors.New("invalid phone number")
	}
	jid = utils.CanonicalJID(jid)

	err = client.MarkRead(context.Background(), data.Id, time.Now(), jid, jid)
	if err != nil {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error marking message as read: %v", instance.Id, err)
		return "", errors.New("error marking message as read")
	}

	return ts.String(), nil
}

func (m *messageService) MarkPlayed(data *MarkPlayedStruct, instance *instance_model.Instance) (string, error) {
	client, err := m.ensureClientConnected(instance.Id)
	if err != nil {
		return "", err
	}

	var ts time.Time
	jID, ok := utils.ParseJID(data.Number)
	if !ok {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error validating message fields", instance.Id)
		return "", errors.New("invalid phone number")
	}
	jID = utils.CanonicalJID(jID)

	err = client.MarkRead(context.Background(), data.Id, time.Now(), jID, jID, types.ReceiptTypePlayed)
	if err != nil {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error marking message as played: %v", instance.Id, err)
		return "", errors.New("error marking message as played")
	}

	return ts.String(), nil
}

func (m *messageService) DownloadMedia(data *DownloadMediaStruct, instance *instance_model.Instance, request *http.Request) (*dataurl.DataURL, string, error) {
	client, err := m.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, "", err
	}

	var ts time.Time

	msg := data.Message

	mimetype := ""
	var mediaData []byte

	img := msg.GetImageMessage()
	audio := msg.GetAudioMessage()
	document := msg.GetDocumentMessage()
	video := msg.GetVideoMessage()
	sticker := msg.GetStickerMessage()

	if img == nil && audio == nil && document == nil && video == nil && sticker == nil {
		return nil, "", errors.New("invalid media type")
	}

	userDirectory := fmt.Sprintf(`files/user_%s`, instance.Id)
	_, err = os.Stat(userDirectory)
	if os.IsNotExist(err) {
		errDir := os.MkdirAll(userDirectory, 0751)
		if errDir != nil {
			m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Could not create user directory (%s)", instance.Id, userDirectory)
			return nil, "", errDir
		}
	}

	if img != nil {
		mediaData, err = client.Download(context.Background(), img)
		if err != nil {
			m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Failed to download image", instance.Id)
			msg := fmt.Sprintf("Failed to download image %v", err)
			return nil, "", errors.New(msg)
		}
		mimetype = img.GetMimetype()
	}

	if audio != nil {
		mediaData, err = client.Download(context.Background(), audio)
		if err != nil {
			m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Failed to download audio", instance.Id)
			msg := fmt.Sprintf("Failed to download audio %v", err)
			return nil, "", errors.New(msg)
		}
		mimetype = audio.GetMimetype()
	}

	if document != nil {
		mediaData, err = client.Download(context.Background(), document)
		if err != nil {
			m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Failed to download document", instance.Id)
			msg := fmt.Sprintf("Failed to download document %v", err)
			return nil, "", errors.New(msg)
		}
		mimetype = document.GetMimetype()
	}

	if video != nil {
		mediaData, err = client.Download(context.Background(), video)
		if err != nil {
			m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Failed to download video", instance.Id)
			msg := fmt.Sprintf("Failed to download video %v", err)
			return nil, "", errors.New(msg)
		}
		mimetype = video.GetMimetype()
	}

	if sticker != nil {
		mediaData, err = client.Download(context.Background(), sticker)
		if err != nil {
			m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Failed to download sticker", instance.Id)
			msg := fmt.Sprintf("Failed to download sticker %v", err)
			return nil, "", errors.New(msg)
		}
		mimetype = sticker.GetMimetype()
	}

	dataURL := dataurl.New(mediaData, mimetype)

	return dataURL, ts.String(), nil
}

func (m *messageService) GetMessageStatus(data *MessageStatusStruct, instance *instance_model.Instance) (*message_model.Message, string, error) {
	_, err := m.ensureClientConnected(instance.Id)
	if err != nil {
		return nil, "", err
	}

	var ts time.Time

	result, err := m.messageRepository.GetMessageByID(data.Id)
	if err != nil {
		return nil, "", err
	}

	return result, ts.String(), nil
}

func (m *messageService) DeleteMessageEveryone(data *MessageStruct, instance *instance_model.Instance) (string, string, error) {
	client, err := m.ensureClientConnected(instance.Id)
	if err != nil {
		return "", "", err
	}

	var ts time.Time

	recipient, ok := utils.ParseJID(data.Chat)
	if !ok {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error validating message fields", instance.Id)
		return "", "", errors.New("invalid phone number")
	}

	m.loggerWrapper.GetLogger(instance.Id).LogInfo("Revoking message %s from %s", data.MessageID, recipient)

	resp, err := client.SendMessage(
		context.Background(),
		recipient,
		client.BuildRevoke(recipient, types.EmptyJID, data.MessageID))
	if err != nil {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error revoking message: %v", instance.Id, err)
		return "", "", err
	}

	response := resp.ID

	return response, ts.String(), nil
}

func (m *messageService) EditMessage(data *EditMessageStruct, instance *instance_model.Instance) (string, string, error) {
	client, err := m.ensureClientConnected(instance.Id)
	if err != nil {
		return "", "", err
	}

	recipient, ok := utils.ParseJID(data.Chat)
	if !ok {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error validating message fields", instance.Id)
		return "", "", errors.New("invalid phone number")
	}

	resp, err := client.SendMessage(
		context.Background(),
		recipient,
		client.BuildEdit(
			recipient,
			data.MessageID,
			&waE2E.Message{
				ExtendedTextMessage: &waE2E.ExtendedTextMessage{
					Text: &data.Message,
				},
			}))
	if err != nil {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error revoking message: %v", instance.Id, err)
		return "", "", err
	}

	return resp.ID, resp.Timestamp.String(), nil
}

func (m *messageService) PinMessage(data *PinMessageStruct, instance *instance_model.Instance) (string, string, error) {
	return m.sendPinMessage(data, instance, true)
}

func (m *messageService) UnpinMessage(data *PinMessageStruct, instance *instance_model.Instance) (string, string, error) {
	return m.sendPinMessage(data, instance, false)
}

func (m *messageService) sendPinMessage(data *PinMessageStruct, instance *instance_model.Instance, pin bool) (string, string, error) {
	client, err := m.ensureClientConnected(instance.Id)
	if err != nil {
		return "", "", err
	}

	chat := data.Chat
	if chat == "" {
		chat = data.Number
	}

	recipient, ok := utils.ParseJID(chat)
	if !ok {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Error validating message fields", instance.Id)
		return "", "", errors.New("invalid chat")
	}

	if data.MessageID == "" {
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Missing messageId in Payload", instance.Id)
		return "", "", errors.New("missing messageId in payload")
	}

	messageKey := &waCommon.MessageKey{
		RemoteJID: proto.String(recipient.String()),
		FromMe:    proto.Bool(data.FromMe),
		ID:        proto.String(data.MessageID),
	}

	if data.Participant != "" {
		participantJID, ok := utils.ParseJID(data.Participant)
		if !ok {
			m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] Invalid participant JID for pin message", instance.Id)
			return "", "", errors.New("invalid participant")
		}
		messageKey.Participant = proto.String(participantJID.String())
	}

	pinType := waE2E.PinInChatMessage_PIN_FOR_ALL
	if !pin {
		pinType = waE2E.PinInChatMessage_UNPIN_FOR_ALL
	}

	msg := &waE2E.Message{
		PinInChatMessage: &waE2E.PinInChatMessage{
			Key:               messageKey,
			Type:              pinType.Enum(),
			SenderTimestampMS: proto.Int64(time.Now().UnixMilli()),
		},
	}

	resp, err := client.SendMessage(context.Background(), recipient, msg)
	if err != nil {
		action := "pin"
		if !pin {
			action = "unpin"
		}
		m.loggerWrapper.GetLogger(instance.Id).LogError("[%s] error sending message %s: %v", instance.Id, action, err)
		return "", "", err
	}

	return resp.ID, resp.Timestamp.String(), nil
}

func NewMessageService(
	clientPointer map[string]*whatsmeow.Client,
	messageRepository message_repository.MessageRepository,
	whatsmeowService whatsmeow_service.WhatsmeowService,
	loggerWrapper *logger_wrapper.LoggerManager,
) MessageService {
	return &messageService{
		clientPointer:     clientPointer,
		messageRepository: messageRepository,
		whatsmeowService:  whatsmeowService,
		loggerWrapper:     loggerWrapper,
	}
}
