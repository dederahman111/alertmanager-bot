package telegram

import (
	"time"

	"github.com/prometheus/alertmanager/template"
	"github.com/tucnak/telebot"
)

// Server saves the exported node
type Server struct {
	id    string
	name  string
	owner string
}

// Member saves the member's telegram info and level in the group
type Member struct {
	username string
	level    string
}

// Members saves all members of chat with level
type Members struct {
	chat    telebot.Chat
	members []Member
}

// HandleAlert shows all of Alert in the
type HandleAlert struct {
	messageID   string
	chat        telebot.Chat
	alert       template.Alert
	level       HandleLevel
	lastUpdate  time.Time
	autoForward bool
}

// HandleLevel shows the level of member is handling the firing alert
type HandleLevel string

const (
	levelOne   HandleLevel = "level1"
	levelTwo   HandleLevel = "level2"
	levelThree HandleLevel = "level3"
)

// BotAlertStore is all the Bot needs to store and read
type BotAlertStore interface {
	List() ([]HandleAlert, error)
	Add(HandleAlert) error
	Remove(HandleAlert) error
}

/* TODO:
 * 		- Command /addserver name owner_id (ex: /addserver nginx 789593887) to monitor
 *		- Response level 1 alert: @owner
 * 		- Response callback FIRING:
 *			+ Ack: Hide all inline buttons, show username of this member, stop auto forward
 *			+ Forward: Show username that did forward the alert and username of Level 2 of recipients, button ‘Forward’ will be hide
 *		- Auto Forward job: The message will be auto forward for next level (Level 2), when the highest level was forwarded, the button *‘Forward’* will be hide.
 *		- Response callback RESOLVED: Hide all buttons of previous alert message, stop auto forward to next Level of previous alert message.
 */

// NewAlert creates the Handle Alert object
func NewAlert(messageID string, chat telebot.Chat, alert template.Alert, bot Bot) (*HandleAlert, error) {
	a := &HandleAlert{
		messageID:   messageID,
		chat:        chat,
		alert:       alert,
		level:       levelOne,
		lastUpdate:  time.Now(),
		autoForward: true,
	}

	return a, nil
}

func (a *HandleAlert) Forward() error {

	return nil
}

func (a *HandleAlert) Acknowledge() error {
	return nil
}

func (a *HandleAlert) AutoForward() error {

	return nil
}

func (a *HandleAlert) callbackHandler(callback *telebot.Callback) error {

}
