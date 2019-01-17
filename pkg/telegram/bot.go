package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/hako/durafmt"
	"github.com/oklog/run"
	"github.com/prometheus/alertmanager/notify"
	"github.com/prometheus/alertmanager/template"
	"github.com/prometheus/alertmanager/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"github.com/tucnak/telebot"
	"github.com/vu-long/alertmanager-bot/pkg/alertmanager"
)

const (
	strAcknowledgeData = "Acknowledge"
	strForwardData     = "Forward"

	commandStart        = "/start"
	commandStop         = "/stop"
	commandHelp         = "/help"
	commandChats        = "/chats"
	commandAddMember    = "/addmember"
	commandRemoveMember = "/rmmember"
	commandMembers      = "/members"
	commandNodes        = "/nodes"

	commandStatus     = "/status"
	commandAlerts     = "/alerts"
	commandSilences   = "/silences"
	commandSilenceAdd = "/silence_add"
	commandSilence    = "/silence"
	commandSilenceDel = "/silence_del"

	responseStart  = "Hey, %s! I will now keep you up to date!\n" + commandHelp
	responseStop   = "Alright, %s! I won't talk to you again.\n" + commandHelp
	responseMember = "Already do your wish!\n"
	responseHelp   = `
I'm a Prometheus AlertManager Bot for Telegram. I will notify you about alerts.
You can also ask me about my ` + commandStatus + `, ` + commandAlerts + ` & ` + commandSilences + `

Available commands:
` + commandStart + ` - Subscribe for alerts.
` + commandStop + ` - Unsubscribe for alerts.
` + commandStatus + ` - Print the current status.
` + commandAlerts + ` - List all alerts.
` + commandSilences + ` - List all silences.
` + commandChats + ` - List all users and group chats that subscribed.
`
)

// BotChatStore is all the Bot needs to store and read
type BotChatStore interface {
	List() ([]telebot.Chat, error)
	Add(telebot.Chat) error
	Remove(telebot.Chat) error
}

// BotMemberStore is all the Bot needs to store and read
type BotMemberStore interface {
	List() ([]Member, error)
	Add(Member) error
	Remove(Member) error
	GetMembersByChat(telebot.Chat) ([]Member, error)
	GetRandomMemberByChatandLevel(telebot.Chat, string) (Member, error)
}

// BotNodeStore is all the Bot needs to store and read
type BotNodeStore interface {
	List() ([]NodeExported, error)
	Add(NodeExported) error
	Remove(NodeExported) error
}

// Bot runs the alertmanager telegram
type Bot struct {
	addr         string
	admins       []int // must be kept sorted
	alertmanager *url.URL
	templates    *template.Template
	chats        BotChatStore
	members      BotMemberStore
	nodes        BotNodeStore
	logger       log.Logger
	revision     string
	startTime    time.Time

	telegram *telebot.Bot

	commandsCounter *prometheus.CounterVec
	webhooksCounter prometheus.Counter
}

// BotOption passed to NewBot to change the default instance
type BotOption func(b *Bot)

// NewBot creates a Bot with the UserStore and telegram telegram
func NewBot(chats BotChatStore, members BotMemberStore, nodes BotNodeStore, token string, admin int, opts ...BotOption) (*Bot, error) {
	bot, err := telebot.NewBot(token)
	if err != nil {
		return nil, err
	}

	commandsCounter := prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "alertmanagerbot",
		Name:      "commands_total",
		Help:      "Number of commands received by command name",
	}, []string{"command"})
	if err := prometheus.Register(commandsCounter); err != nil {
		return nil, err
	}

	b := &Bot{
		logger:          log.NewNopLogger(),
		telegram:        bot,
		chats:           chats,
		members:         members,
		nodes:           nodes,
		addr:            "127.0.0.1:8080",
		admins:          []int{admin},
		alertmanager:    &url.URL{Host: "localhost:9093"},
		commandsCounter: commandsCounter,
		// TODO: initialize templates with default?
	}

	for _, opt := range opts {
		opt(b)
	}

	return b, nil
}

// WithLogger sets the logger for the Bot as an option
func WithLogger(l log.Logger) BotOption {
	return func(b *Bot) {
		b.logger = l
	}
}

// WithAddr sets the internal listening addr of the bot's web server receiving webhooks
func WithAddr(addr string) BotOption {
	return func(b *Bot) {
		b.addr = addr
	}
}

// WithAlertmanager sets the connection url for the Alertmanager
func WithAlertmanager(u *url.URL) BotOption {
	return func(b *Bot) {
		b.alertmanager = u
	}
}

// WithTemplates uses Alertmanager template to render messages for Telegram
func WithTemplates(t *template.Template) BotOption {
	return func(b *Bot) {
		b.templates = t
	}
}

// WithRevision is setting the Bot's revision for status commands
func WithRevision(r string) BotOption {
	return func(b *Bot) {
		b.revision = r
	}
}

// WithStartTime is setting the Bot's start time for status commands
func WithStartTime(st time.Time) BotOption {
	return func(b *Bot) {
		b.startTime = st
	}
}

// WithExtraAdmins allows the specified additional user IDs to issue admin
// commands to the bot.
func WithExtraAdmins(ids ...int) BotOption {
	return func(b *Bot) {
		b.admins = append(b.admins, ids...)
		sort.Ints(b.admins)
	}
}

// SendAdminMessage to the admin's ID with a message
func (b *Bot) SendAdminMessage(adminID int, message string) {
	b.telegram.SendMessage(telebot.User{ID: adminID}, message, nil)
}

// isAdminID returns whether id is one of the configured admin IDs.
func (b *Bot) isAdminID(id int) bool {
	i := sort.SearchInts(b.admins, id)
	return i < len(b.admins) && b.admins[i] == id
}

// Run the telegram and listen to messages send to the telegram
func (b *Bot) Run(ctx context.Context, webhooks <-chan notify.WebhookMessage) error {
	commandSuffix := fmt.Sprintf("@%s", b.telegram.Identity.Username)

	commands := map[string]func(message telebot.Message){
		commandStart:        b.handleStart,
		commandStop:         b.handleStop,
		commandHelp:         b.handleHelp,
		commandChats:        b.handleChats,
		commandStatus:       b.handleStatus,
		commandAlerts:       b.handleAlerts,
		commandSilences:     b.handleSilences,
		commandAddMember:    b.handleAddMember,
		commandRemoveMember: b.handleRemoveMember,
		commandMembers:      b.handleMembers,
		commandNodes:        b.handleNodes,
	}

	// init counters with 0
	for command := range commands {
		b.commandsCounter.WithLabelValues(command).Add(0)
	}

	process := func(message telebot.Message) error {
		if message.IsService() {
			return nil
		}

		if !b.isAdminID(message.Sender.ID) {
			b.commandsCounter.WithLabelValues("dropped").Inc()
			return fmt.Errorf("dropped message from forbidden sender")
		}

		if err := b.telegram.SendChatAction(message.Chat, telebot.Typing); err != nil {
			return err
		}

		// Remove the command suffix from the text, /help@BotName => /help
		text := strings.Replace(message.Text, commandSuffix, "", -1)
		// Only take the first part into account, /help foo => /help
		text = strings.Split(text, " ")[0]

		level.Debug(b.logger).Log("msg", "message received", "text", text)

		// Get the corresponding handler from the map by the commands text
		handler, ok := commands[text]

		if !ok {
			b.commandsCounter.WithLabelValues("incomprehensible").Inc()
			b.telegram.SendMessage(
				message.Chat,
				"Sorry, I don't understand...",
				nil,
			)
			return nil
		}

		b.commandsCounter.WithLabelValues(text).Inc()
		handler(message)

		return nil
	}

	messages := make(chan telebot.Message, 100)
	queries := make(chan telebot.Query, 500)
	callbacks := make(chan telebot.Callback, 500)
	b.telegram.Messages = messages
	b.telegram.Queries = queries
	b.telegram.Callbacks = callbacks
	// b.telegram.Listen(messages, time.Second)
	go b.telegram.Start(1 * time.Second)
	alertchan := make(chan *HandleAlert, 100)

	var gr run.Group
	{
		gr.Add(func() error {
			return b.sendWebhook(ctx, webhooks, alertchan)
		}, func(err error) {
		})
	}
	{
		gr.Add(func() error {
			// var HandleAlerts []HandleAlert
			HandleAlerts := make(map[string][]*HandleAlert)
			for {
				select {
				case <-ctx.Done():
					return nil
				case message := <-messages:
					if err := process(message); err != nil {
						level.Info(b.logger).Log(
							"msg", "failed to process message",
							"err", err,
							"sender_id", message.Sender.ID,
							"sender_username", message.Sender.Username,
						)
					}
				case callback := <-callbacks:
					level.Debug(b.logger).Log(
						"msg", "received callback",
						"data", callback.Data,
						"sender_id", callback.Sender.ID,
						"sender_username", callback.Sender.Username,
						"message_id", callback.Message.ID,
					)

					var cd CallbackData
					dec := json.NewDecoder(strings.NewReader(callback.Data))
					dec.Decode(&cd)
					if err := dec.Decode(&cd); err == io.EOF {
						// TODO: Handle this case
					} else if err != nil {
						level.Error(b.logger).Log("err", err)
					}

					// Handle if member press the "Acknowledge" button
					if cd.Button == strAcknowledgeData {
						for _, h := range HandleAlerts[cd.AlertID] {
							level.Debug(b.logger).Log(
								"msg", "run Acknowledge at",
								"data", h.ID,
							)
							h.MessageID = callback.Message.ID
							err := h.Acknowledge(b.telegram, callback)
							if err != nil {
								level.Error(b.logger).Log(
									"msg", "failed to acknowledge",
									"err", err,
								)
							}
						}
					} else if cd.Button == strForwardData {
						// Handle if member press the "Forward" button
						for _, h := range HandleAlerts[cd.AlertID] {
							level.Debug(b.logger).Log(
								"msg", "run Forward at",
								"data", h.ID,
							)
							h.MessageID = callback.Message.ID
							ackData, err := NewCallbackData(strAcknowledgeData, h.ID)
							if err != nil {
								break
							}
							jsonAckStr, err := json.Marshal(ackData)
							err = h.Forward(b.telegram, callback, string(jsonAckStr))
							if err != nil {
								level.Error(b.logger).Log(
									"msg", "failed to forward",
									"err", err,
								)
							}
						}

					}
				case a := <-alertchan:
					// Get the HandleAlert in channel and save
					// HandleAlerts = append(HandleAlerts, a)
					if HandleAlerts[a.ID] == nil {
						HandleAlerts[a.ID] = append(HandleAlerts[a.ID], a)
						level.Debug(b.logger).Log(
							"msg", "received alert",
							"data", a.ID,
						)
					}
				}

			}
		}, func(err error) {
		})
	}

	return gr.Run()
}

// sendWebhook sends messages received via webhook to all subscribed chats
func (b *Bot) sendWebhook(ctx context.Context, webhooks <-chan notify.WebhookMessage, alerts chan<- *HandleAlert) error {
	HandleAlerts := make(map[string][]*HandleAlert)
	for {
		select {
		case <-ctx.Done():
			return nil
		case w := <-webhooks:
			chats, err := b.chats.List()
			if err != nil {
				level.Error(b.logger).Log("msg", "failed to get chat list from store", "err", err)
				continue
			}

			data := &template.Data{
				Receiver:          w.Receiver,
				Status:            w.Status,
				Alerts:            w.Alerts,
				GroupLabels:       w.GroupLabels,
				CommonLabels:      w.CommonLabels,
				CommonAnnotations: w.CommonAnnotations,
				ExternalURL:       w.ExternalURL,
			}

			out, err := b.templates.ExecuteHTMLString(`{{ template "telegram.default" . }}`, data)
			if err != nil {
				level.Warn(b.logger).Log("msg", "failed to template alerts", "err", err)
				continue
			}

			id := data.GroupLabels.Values()[0]
			for _, chat := range chats {
				// If receive the resolved signal via webhook, Resolve() all of HandlerAlert in the map list
				if w.Status == string(model.AlertResolved) {
					// TODO: If do not have any reaction from member, maybe having bug here because HandleAlert.MessageID is nil
					if HandleAlerts[id] != nil {
						for _, h := range HandleAlerts[id] {
							h.Resolved(b.telegram)
						}
					}
				} else if w.Status == string(model.AlertFiring) {
					// If receive the firing signal via webhook, create the inline message with 2 buttons,
					ackData, err := NewCallbackData(strAcknowledgeData, id)
					if err != nil {
						break
					}
					jsonAckStr, err := json.Marshal(ackData)
					if err != nil {
						break
					}
					level.Debug(b.logger).Log("json", jsonAckStr)
					fwdData, err := NewCallbackData(strForwardData, id)
					if err != nil {
						break
					}
					jsonFwdStr, err := json.Marshal(fwdData)
					if err != nil {
						break
					}
					level.Debug(b.logger).Log("json", jsonFwdStr)

					b.telegram.SendMessage(chat, out, &telebot.SendOptions{
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

					// And create new HandleAlert object and put it to channel
					alert, err := NewAlert(id, chat, data.Alerts[0], *b)
					if err != nil {
						level.Error(b.logger).Log("msg", "failed to create new handle alert", "err", err)
						break
					}
					alerts <- alert

					// Save it to process whenever receive resolved signal
					HandleAlerts[alert.ID] = append(HandleAlerts[alert.ID], alert)
				}
			}
		}

	}
}

func (b *Bot) handleStart(message telebot.Message) {
	if err := b.chats.Add(message.Chat); err != nil {
		level.Warn(b.logger).Log("msg", "failed to add chat to chat store", "err", err)
		b.telegram.SendMessage(message.Chat, "I can't add this chat to the subscribers list.", nil)
		return
	}

	b.telegram.SendMessage(message.Chat, fmt.Sprintf(responseStart, message.Sender.FirstName), nil)
	level.Info(b.logger).Log(
		"user subscribed",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)
}

func (b *Bot) handleStop(message telebot.Message) {
	if err := b.chats.Remove(message.Chat); err != nil {
		level.Warn(b.logger).Log("msg", "failed to remove chat from chat store", "err", err)
		b.telegram.SendMessage(message.Chat, "I can't remove this chat from the subscribers list.", nil)
		return
	}

	b.telegram.SendMessage(message.Chat, fmt.Sprintf(responseStop, message.Sender.FirstName), nil)
	level.Info(b.logger).Log(
		"user unsubscribed",
		"username", message.Sender.Username,
		"user_id", message.Sender.ID,
	)
}

func (b *Bot) handleHelp(message telebot.Message) {
	b.telegram.SendMessage(message.Chat, responseHelp, nil)
}

func (b *Bot) handleChats(message telebot.Message) {
	chats, err := b.chats.List()
	if err != nil {
		level.Warn(b.logger).Log("msg", "failed to list chats from chat store", "err", err)
		b.telegram.SendMessage(message.Chat, "I can't list the subscribed chats.", nil)
		return
	}

	list := ""
	for _, chat := range chats {
		if chat.IsGroupChat() {
			list = list + fmt.Sprintf("@%s\n", chat.Title)
		} else {
			list = list + fmt.Sprintf("@%s\n", chat.Username)
		}
	}

	b.telegram.SendMessage(message.Chat, "Currently these chat have subscribed:\n"+list, nil)
}

func (b *Bot) handleStatus(message telebot.Message) {
	s, err := alertmanager.Status(b.logger, b.alertmanager.String())
	if err != nil {
		level.Warn(b.logger).Log("msg", "failed to get status", "err", err)
		b.telegram.SendMessage(message.Chat, fmt.Sprintf("failed to get status... %v", err), nil)
		return
	}

	uptime := durafmt.Parse(time.Since(s.Data.Uptime))
	uptimeBot := durafmt.Parse(time.Since(b.startTime))

	b.telegram.SendMessage(
		message.Chat,
		fmt.Sprintf(
			"*AlertManager*\nVersion: %s\nUptime: %s\n*AlertManager Bot*\nVersion: %s\nUptime: %s",
			s.Data.VersionInfo.Version,
			uptime,
			b.revision,
			uptimeBot,
		),
		&telebot.SendOptions{ParseMode: telebot.ModeMarkdown},
	)
}

func (b *Bot) handleAlerts(message telebot.Message) {
	alerts, err := alertmanager.ListAlerts(b.logger, b.alertmanager.String())
	if err != nil {
		b.telegram.SendMessage(message.Chat, fmt.Sprintf("failed to list alerts... %v", err), nil)
		return
	}

	if len(alerts) == 0 {
		b.telegram.SendMessage(message.Chat, "No alerts right now! 🎉", nil)
		return
	}

	out, err := b.tmplAlerts(alerts...)
	if err != nil {
		return
	}

	err = b.telegram.SendMessage(message.Chat, out, &telebot.SendOptions{
		ParseMode: telebot.ModeHTML,
	})
	if err != nil {
		level.Warn(b.logger).Log("msg", "failed to send message", "err", err)
	}
}

func (b *Bot) handleSilences(message telebot.Message) {
	silences, err := alertmanager.ListSilences(b.logger, b.alertmanager.String())
	if err != nil {
		b.telegram.SendMessage(message.Chat, fmt.Sprintf("failed to list silences... %v", err), nil)
		return
	}

	if len(silences) == 0 {
		b.telegram.SendMessage(message.Chat, "No silences right now.", nil)
		return
	}

	var out string
	for _, silence := range silences {
		out = out + alertmanager.SilenceMessage(silence) + "\n"
	}

	b.telegram.SendMessage(message.Chat, out, &telebot.SendOptions{ParseMode: telebot.ModeMarkdown})
}

func (b *Bot) tmplAlerts(alerts ...*types.Alert) (string, error) {
	data := b.templates.Data("default", nil, alerts...)

	out, err := b.templates.ExecuteHTMLString(`{{ template "telegram.default" . }}`, data)
	if err != nil {
		return "", err
	}

	return out, nil
}

func (b *Bot) handleAddMember(message telebot.Message) {
	// Right format: '/addmember username level (node if level = 1)'.
	// Ex: /addmember vu_long 1 httpd
	params := strings.Split(message.Text, " ")
	if len(params) < 3 || len(params) > 4 {
		level.Warn(b.logger).Log("msg", "need 2-3 parameters")
		b.telegram.SendMessage(message.Chat, "Please send right format: '/addmember username level (node if level = 1)'. Ex: /addmember vu_long 1 httpd", nil)
		return
	}

	if HandleLevel(params[2]) != levelOne && HandleLevel(params[2]) != levelTwo && HandleLevel(params[2]) != levelThree {
		level.Warn(b.logger).Log("msg", "level need to be 1-3")
		b.telegram.SendMessage(message.Chat, "Level need to be \"[1-3]\"", nil)
		return
	}

	member := Member{
		Username: params[1],
		Level:    HandleLevel(params[2]),
		Chat:     message.Chat,
	}

	if err := b.members.Add(member); err != nil {
		level.Warn(b.logger).Log("msg", "failed to add chat to chat store", "err", err)
		b.telegram.SendMessage(message.Chat, "I can't add this member to the subscribers list.", nil)
		return
	}

	if member.Level == levelOne {
		node := NodeExported{
			Name:  params[3],
			Owner: member.Username,
		}

		if err := b.nodes.Add(node); err != nil {
			level.Warn(b.logger).Log("msg", "failed to add node exported to node store", "err", err)
			b.telegram.SendMessage(message.Chat, "I can't add this node to the subscribers list.", nil)
			return
		}
	}

	b.telegram.SendMessage(message.Chat, responseMember, nil)
	level.Info(b.logger).Log(
		"msg", "Member added",
		"username", member.Username,
		"level", member.Level,
	)
}

func (b *Bot) handleRemoveMember(message telebot.Message) {
	// Right format: '/rmmember username level (node if level = 1)'.
	// Ex: /rmmember vu_long 1 httpd
	params := strings.Split(message.Text, " ")
	level.Debug(b.logger).Log("len", len(params))
	if len(params) != 2 {
		level.Warn(b.logger).Log("msg", "need only 1 parameter")
		b.telegram.SendMessage(message.Chat, "Please send right format: '/rmmember username'. Ex: /rmmember vu_long", nil)
		return
	}

	member := Member{
		Username: params[1],
	}

	if err := b.members.Remove(member); err != nil {
		level.Warn(b.logger).Log("msg", "failed to remove chat to chat store", "err", err)
		b.telegram.SendMessage(message.Chat, "I can't remove this member to the subscribers list.", nil)
		return
	}

	b.telegram.SendMessage(message.Chat, responseMember, nil)
	level.Info(b.logger).Log(
		"msg", "Member is removed",
		"username", member.Username,
	)
}

func (b *Bot) handleMembers(message telebot.Message) {
	members, err := b.members.List()
	if err != nil {
		level.Warn(b.logger).Log("msg", "failed to list members from member store", "err", err)
		b.telegram.SendMessage(message.Chat, "I can't list the added members.", nil)
		return
	}

	list := ""
	for _, member := range members {
		list = list + fmt.Sprintf("@%s level: %s\n", member.Username, member.Level)
	}

	level.Debug(b.logger).Log("list", list)

	b.telegram.SendMessage(message.Chat, "Currently these members have added:\n"+list, nil)
}

func (b *Bot) handleNodes(message telebot.Message) {
	nodes, err := b.nodes.List()
	if err != nil {
		level.Warn(b.logger).Log("msg", "failed to list members from nodes store", "err", err)
		b.telegram.SendMessage(message.Chat, "I can't list the added nodes.", nil)
		return
	}

	list := ""
	for _, node := range nodes {
		list = list + fmt.Sprintf("@%s level: %s\n", node.Name, node.Owner)
	}

	level.Debug(b.logger).Log("list", list)

	b.telegram.SendMessage(message.Chat, "Currently these nodes have added:\n"+list, nil)
}
