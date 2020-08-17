package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	ubot "github.com/UBotPlatform/UBot.Common.Go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var event *ubot.AccountEventEmitter
var bot *tgbotapi.BotAPI

func getGroupName(id string) (string, error) {
	chat, err := bot.GetChat(tgbotapi.ChatConfig{SuperGroupUsername: id})
	if err != nil {
		return "", err
	}
	return chat.Title, nil
}
func getUserName(id string) (string, error) {
	return "", errors.New("not supported")
}
func login() error {
	return errors.New("not supported")
}
func logout() error {
	return errors.New("not supported")
}
func markdownEscaped(s string) string {
	data := strings.ReplaceAll(s, `\`, `\\`)
	data = strings.ReplaceAll(data, `*`, `\*`)
	data = strings.ReplaceAll(data, `_`, `\_`)
	data = strings.ReplaceAll(data, `~`, `\~`)
	data = strings.ReplaceAll(data, "`", "\\`")
	data = strings.ReplaceAll(data, `[`, `\[`)
	data = strings.ReplaceAll(data, `]`, `\]`)
	data = strings.ReplaceAll(data, `(`, `\(`)
	data = strings.ReplaceAll(data, `)`, `\)`)
	return data
}
func sendChatMessage(msgType ubot.MsgType, source string, target string, message string) error {
	iSource, _ := strconv.ParseInt(source, 10, 64)
	entities := ubot.ParseMsg(message)
	var rawMsg strings.Builder
	packets := make([]tgbotapi.Chattable, 0, 1)
	for _, entity := range entities {
		switch entity.Type {
		case "image_online":
			if rawMsg.Len() != 0 {
				msg := tgbotapi.NewMessageToChannel(source, rawMsg.String())
				msg.ParseMode = "MarkdownV2"
				packets = append(packets, msg)
				rawMsg.Reset()
			}
			resp, err := http.Get(entity.Data)
			if err != nil {
				continue
			}
			defer resp.Body.Close()
			bytes, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				continue
			}
			photoMsg := tgbotapi.NewPhotoUpload(iSource, tgbotapi.FileBytes{
				Name:  "ubot_image.png",
				Bytes: bytes})
			packets = append(packets, photoMsg)
		case "text":
			rawMsg.WriteString(markdownEscaped(entity.Data))
		case "at":
			if len(entity.Data) > 0 && entity.Data[0] == '@' {
				rawMsg.WriteString(entity.Data)
				continue
			}
			atUser, err := strconv.Atoi(target)
			if err != nil {
				continue
			}
			cm, err := bot.GetChatMember(tgbotapi.ChatConfigWithUser{
				SuperGroupUsername: source,
				UserID:             atUser})
			if err != nil {
				continue
			}
			rawMsg.WriteString("[")
			rawMsg.WriteString(cm.User.FirstName)
			rawMsg.WriteString("](tg://user?id=")
			rawMsg.WriteString(fmt.Sprint(cm.User.ID))
			rawMsg.WriteString(")")
		default:
			rawMsg.WriteString("[不支持的消息类型]")
		}
	}
	if rawMsg.Len() != 0 {
		msg := tgbotapi.NewMessageToChannel(source, rawMsg.String())
		msg.ParseMode = "MarkdownV2"
		packets = append(packets, msg)
	}
	for _, packet := range packets {
		_, err := bot.Send(packet)
		if err != nil {
			return err
		}
	}
	return nil
}
func removeMember(source string, target string) error {
	var err error
	var config tgbotapi.KickChatMemberConfig
	config.SuperGroupUsername = source
	config.UserID, err = strconv.Atoi(target)
	if err != nil {
		return err
	}
	resp, err := bot.KickChatMember(config)
	if err != nil {
		return err
	}
	if !resp.Ok {
		return errors.New(resp.Description)
	}
	return nil
}
func shutupMember(source string, target string, duration int) error {
	return errors.New("not supported")
}
func shutupAllMember(source string, shutupSwitch bool) error {
	return errors.New("not supported")
}

func getMemberName(source string, target string) (string, error) {
	iTarget, err := strconv.Atoi(target)
	if err != nil {
		return "", err
	}
	cm, err := bot.GetChatMember(tgbotapi.ChatConfigWithUser{SuperGroupUsername: source, UserID: iTarget})
	if err != nil {
		return "", err
	}
	if cm.User.LastName != "" {
		return fmt.Sprintf("%s %s", cm.User.FirstName, cm.User.LastName), nil
	}
	return cm.User.FirstName, nil
}

func getUserAvatar(id string) (string, error) {
	iID, err := strconv.Atoi(id)
	if err != nil {
		return "", err
	}
	upp, err := bot.GetUserProfilePhotos(tgbotapi.UserProfilePhotosConfig{UserID: iID})
	if err != nil {
		return "", err
	}
	if len(upp.Photos) == 0 {
		return "", nil
	}
	if len(upp.Photos[0]) == 0 {
		return "", nil
	}
	fid := upp.Photos[0][len(upp.Photos[0])-1].FileID
	photoFile, err := bot.GetFile(tgbotapi.FileConfig{FileID: fid})
	if err != nil {
		return "", err
	}
	return photoFile.Link(bot.Token), nil
}

func getSelfID() (string, error) {
	return fmt.Sprint(bot.Self.ID), nil
}

type tgEntitiesInUTF8 struct {
	ubot.MsgEntity
	Start int
	End   int
}

func receiveTGMessage(message *tgbotapi.Message) {
	if message.NewChatMembers != nil {
		for _, member := range *message.NewChatMembers {
			_ = event.OnMemberJoined(fmt.Sprint(message.Chat.ID), fmt.Sprint(member.ID), "")
		}
		return
	}
	if message.LeftChatMember != nil {
		_ = event.OnMemberLeft(fmt.Sprint(message.Chat.ID), fmt.Sprint(message.LeftChatMember.ID))
		return
	}
	if message.From == nil || message.From.ID == bot.Self.ID {
		return
	}
	myEntities := make([]*tgEntitiesInUTF8, 0, 1)
	msgType := ubot.GroupMsg
	if message.Chat.IsPrivate() {
		msgType = ubot.PrivateMsg
	}
	if message.Photo != nil {
		photos := *message.Photo
		photoFile, err := bot.GetFile(tgbotapi.FileConfig{FileID: photos[len(photos)-1].FileID})
		if err == nil {
			myEntities = append(myEntities, &tgEntitiesInUTF8{
				MsgEntity: ubot.MsgEntity{Type: "image_online", Data: photoFile.Link(bot.Token)},
				Start:     0,
				End:       0})
		}
	}
	msgText := message.Text
	entities := message.Entities
	if msgText == "" {
		msgText = message.Caption
	}
	curUTF16Pos := 0
	curBytePos := 0
	if entities != nil && len(*entities) > 0 {
		for _, entity := range *entities {
			if curUTF16Pos > entity.Offset {
				curBytePos = 0
				curUTF16Pos = 0
			}
			for curUTF16Pos < entity.Offset {
				r, width := utf8.DecodeRuneInString(msgText[curBytePos:])
				curBytePos += width
				curUTF16Pos += len(utf16.Encode([]rune{r}))
			}
			if curUTF16Pos > entity.Offset {
				continue
			}
			start := curUTF16Pos
			endInUTF16 := entity.Offset + entity.Length
			for curUTF16Pos < endInUTF16 {
				r, width := utf8.DecodeRuneInString(msgText[curBytePos:])
				curBytePos += width
				curUTF16Pos += len(utf16.Encode([]rune{r}))
			}
			if curUTF16Pos > endInUTF16 {
				continue
			}
			end := curBytePos
			switch entity.Type {
			case "mention":
				username := msgText[start:end]
				myEntities = append(myEntities, &tgEntitiesInUTF8{
					MsgEntity: ubot.MsgEntity{Type: "at", Data: username},
					Start:     start,
					End:       end})
			case "text_mention":
				if entity.User == nil {
					continue
				}
				myEntities = append(myEntities, &tgEntitiesInUTF8{
					MsgEntity: ubot.MsgEntity{Type: "at", Data: fmt.Sprint(entity.User.ID)},
					Start:     start,
					End:       end})
			}
		}
	}
	var builder ubot.MsgBuilder
	lastFinished := 0
	for _, myEntity := range myEntities {
		builder.WriteString(msgText[lastFinished:myEntity.Start])
		builder.WriteEntity(myEntity.MsgEntity)
		lastFinished = myEntity.End
	}
	builder.WriteString(msgText[lastFinished:])
	ubotMsg := builder.String()
	if ubotMsg == "" {
		return
	}
	_ = event.OnReceiveChatMessage(msgType,
		fmt.Sprint(message.Chat.ID),
		fmt.Sprint(message.From.ID),
		ubotMsg,
		ubot.MsgInfo{ID: fmt.Sprint(message.MessageID)})
}
func receiveTGUpdates() {
	u := tgbotapi.NewUpdate(-1)
	u.Timeout = 60
	updates, err := bot.GetUpdatesChan(u)
	if err != nil {
		panic(errors.New("Failed to get updates channel: " + err.Error()))
	}
	for update := range updates {
		switch {
		case update.Message != nil:
			go receiveTGMessage(update.Message)
		}
	}
}

func main() {
	var err error
	bot, err = tgbotapi.NewBotAPI(os.Args[3])
	ubot.AssertNoError(err)
	err = ubot.HostAccount("Telegram Bot", func(e *ubot.AccountEventEmitter) *ubot.Account {
		event = e
		go receiveTGUpdates()
		return &ubot.Account{
			GetGroupName:    getGroupName,
			GetUserName:     getUserName,
			Login:           login,
			Logout:          logout,
			SendChatMessage: sendChatMessage,
			RemoveMember:    removeMember,
			ShutupMember:    shutupMember,
			ShutupAllMember: shutupAllMember,
			GetMemberName:   getMemberName,
			GetUserAvatar:   getUserAvatar,
			GetSelfID:       getSelfID,
		}
	})
	ubot.AssertNoError(err)
	_ = logout()
}
