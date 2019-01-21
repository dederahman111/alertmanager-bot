package telegram

// CallbackData save the json struct to communication in inline button data
type CallbackData struct {
	Button  string `json:"button"`
	AlertID string `json:"alert"`
}

// NewCallbackData create new CallbackData object
func NewCallbackData(button string, alert string) (*CallbackData, error) {
	return &CallbackData{
		Button:  button,
		AlertID: alert,
	}, nil
}
