package telegram

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/prometheus/alertmanager/template"
	"github.com/tucnak/telebot"
)

// HandleAlert shows all of Alert in the
type HandleAlert struct {
	ID              string
	MessageID       int
	MemberStore     BotMemberStore
	NodeStore       BotNodeStore
	Chat            telebot.Chat
	Alert           template.Alert
	Level           HandleLevel
	LastUpdate      time.Time
	AutoForwardFlag bool
}

// Destination is internal inline message ID.
func (a HandleAlert) Destination() string {
	return strconv.Itoa(a.MessageID)
}

// HandleLevel shows the level of member is handling the firing alert
type HandleLevel string

const (
	levelOne   HandleLevel = "1"
	levelTwo   HandleLevel = "2"
	levelThree HandleLevel = "3"

	// AutoForwardTimeout If no one action that message in 5 minutes, then do auto forward
	AutoForwardTimeout time.Duration = 5 * time.Minute

	strAcknowledge string = "Acknowledge by: @%s"
	strForward     string = "@%s forward to @%s"
	strAutoForward string = "Auto forward to next level @%s"
)

// BotAlertStore is all the Bot needs to store and read
type BotAlertStore interface {
	List() ([]HandleAlert, error)
	Add(HandleAlert) error
	Remove(HandleAlert) error
}

/* TODO:
 * 		v - Command /addserver name owner_id (ex: /addserver nginx 789593887) to monitor
 *		- Response level 1 alert: @owner
 * 		v Response callback FIRING:
 *		v	+ Ack: Hide all inline buttons, show username of this member, stop auto forward
 *		v	+ Forward: Show username that did forward the alert and username of Level 2 of recipients, button ‘Forward’ will be hide
 *		v Auto Forward job: The message will be auto forward for next level (Level 2), when the highest level was forwarded, the button *‘Forward’* will be hide.
 *		v Response callback RESOLVED: Hide all buttons of previous alert message, stop auto forward to next Level of previous alert message.
 */

// NewAlert creates the Handle Alert object
func NewAlert(id string, chat telebot.Chat, alert template.Alert, b *Bot, out string) (*HandleAlert, error) {
	// Prepare source to send the message
	ackData, err := NewCallbackData(strAcknowledgeData, id)
	if err != nil {
		return nil, err
	}
	jsonAckStr, err := json.Marshal(ackData)
	if err != nil {
		return nil, err
	}
	fwdData, err := NewCallbackData(strForwardData, id)
	if err != nil {
		return nil, err
	}
	jsonFwdStr, err := json.Marshal(fwdData)
	if err != nil {
		return nil, err
	}

	respMsg, err := b.telegram.SendMessage(chat, out, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
		ReplyMarkup: telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.KeyboardButton{
				[]telebot.KeyboardButton{
					telebot.KeyboardButton{
						Text: strAcknowledgeData,
						Data: string(jsonAckStr), // Callback query
					},
					telebot.KeyboardButton{
						Text: strForwardData,
						Data: string(jsonFwdStr), // Callback query
					},
				},
			},
		},
	})
	if err != nil {
		return nil, err
	}
	level.Debug(b.logger).Log("message_id", strconv.Itoa(respMsg.ID))

	a := &HandleAlert{
		ID:              id,
		MessageID:       respMsg.ID,
		MemberStore:     b.members,
		NodeStore:       b.nodes,
		Chat:            chat,
		Alert:           alert,
		Level:           levelOne,
		LastUpdate:      time.Now(),
		AutoForwardFlag: true,
	}

	nodes, err := a.NodeStore.List()
	if err != nil {
		return nil, err
	}
	memberID := ""
	for _, n := range nodes {
		if n.Name == a.ID {
			memberID = n.Owner
		}
	}
	if memberID == "" {
		randMember, err := a.MemberStore.GetRandomMemberByChatandLevel(a.Chat, string(a.Level))
		if err != nil {
			return nil, err
		}
		memberID = randMember.Username
	}
	go a.AutoForward(b.telegram, 5*time.Second)

	respString := fmt.Sprintf("@%s", memberID)
	_, err = b.telegram.SendMessage(a.Chat, respString, nil)
	if err != nil {
		return nil, err
	}

	return a, nil
}

// Acknowledge is function to process callback whenever member press the Acknowledge button
func (a *HandleAlert) Acknowledge(bot *telebot.Bot, callback telebot.Callback) error {
	a.AutoForwardFlag = false

	respString := fmt.Sprintf(strAcknowledge, callback.Sender.Username)
	_, err := bot.SendMessage(a.Chat, respString, nil)
	if err != nil {
		return err
	}
	err = bot.EditMessageReplyMakeup(a.Chat, a.MessageID, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	})
	if err != nil {
		return err
	}

	return nil
}

// Forward is function to process callback whenever member press the Forward button
func (a *HandleAlert) Forward(bot *telebot.Bot, callback telebot.Callback, data string) error {
	a.IncreaseLevel()
	randMember, err := a.MemberStore.GetRandomMemberByChatandLevel(a.Chat, string(a.Level))
	if err != nil {
		return err
	}

	respString := fmt.Sprintf(strForward, callback.Sender.Username, randMember.Username)
	_, err = bot.SendMessage(a.Chat, respString, nil)
	if err != nil {
		return err
	}

	err = bot.EditMessageReplyMakeup(a.Chat, a.MessageID, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
		ReplyMarkup: telebot.ReplyMarkup{
			InlineKeyboard: [][]telebot.KeyboardButton{
				[]telebot.KeyboardButton{
					telebot.KeyboardButton{
						Text: strAcknowledgeData,
						Data: data, // Callback query
					},
				},
			},
		},
	})
	if err != nil {
		return err
	}
	return nil
}

// AutoForward job run to auto forward and push the alert to telegram alert group
func (a *HandleAlert) AutoForward(bot *telebot.Bot, timeout time.Duration) error {
	for a.AutoForwardFlag == true {
		if time.Since(a.LastUpdate) >= AutoForwardTimeout {
			a.LastUpdate = time.Now()
			a.IncreaseLevel()
			randMember, err := a.MemberStore.GetRandomMemberByChatandLevel(a.Chat, string(a.Level))
			if err != nil {
				return err
			}

			respString := fmt.Sprintf(strAutoForward, randMember.Username)
			bot.SendMessage(a.Chat, respString, nil)
		}
		// Wait for a bit and try again.
		time.Sleep(timeout)
	}

	return nil
}

// Resolved handle resolve signal from callback
func (a *HandleAlert) Resolved(bot *telebot.Bot, out string) error {
	a.AutoForwardFlag = false
	_, err := bot.SendMessage(a.Chat, out, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	})
	if err != nil {
		return err
	}

	err = bot.EditMessageReplyMakeup(a.Chat, a.MessageID, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	})
	if err != nil {
		return err
	}
	return nil
}

// IncreaseLevel increase the level on alert
func (a *HandleAlert) IncreaseLevel() bool {
	if a.Level == levelOne {
		a.Level = levelTwo
	} else if a.Level == levelTwo {
		a.Level = levelThree
	} else {
		return false
	}
	return true
}

func (a *HandleAlert) callbackHandler(callback telebot.Callback) error {
	return nil
}
