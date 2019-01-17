package telegram

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
