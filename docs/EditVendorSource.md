## Telebot.v1

#### bot.go

'''
// EditMessageReplyMakeup edits reply makeup of a message.
func (b *Bot) EditMessageReplyMakeup(recipient Recipient, options *SendOptions) error {
	params := map[string]string{
		"inline_message_id": recipient.Destination(),
	}

	if options != nil {
		embedSendOptions(params, options)
	}

	responseJSON, err := b.sendCommand("editMessageReplyMarkup", params)
	if err != nil {
		return err
	}

	var responseReceived struct {
		Ok          bool
		Description string
	}

	err = json.Unmarshal(responseJSON, &responseReceived)
	if err != nil {
		return errors.Wrap(err, "bad response json")
	}

	if !responseReceived.Ok {
		return errors.Errorf("api error: %s", responseReceived.Description)
	}

	return nil
}
'''