package telebot

import (
	"encoding/json"

	"github.com/pkg/errors"
)

func extractMsgResponse(respJSON []byte) (*Message, error) {
	var resp struct {
		Ok          bool
		Result      *Message
		Description string
	}

	err := json.Unmarshal(respJSON, &resp)
	if err != nil {
		var resp struct {
			Ok          bool
			Result      bool
			Description string
		}

		err := json.Unmarshal(respJSON, &resp)
		if err != nil {
			return nil, errors.Wrap(err, "bad response json")
		}

		if !resp.Ok {
			return nil, errors.Errorf("api error: %s", resp.Description)
		}
	}

	if !resp.Ok {
		return nil, errors.Errorf("api error: %s", resp.Description)
	}

	return resp.Result, nil
}

func extractOkResponse(respJSON []byte) error {
	var resp struct {
		Ok          bool
		Description string
	}

	err := json.Unmarshal(respJSON, &resp)
	if err != nil {
		return errors.Wrap(err, "bad response json")
	}

	if !resp.Ok {
		return errors.Errorf("api error: %s", resp.Description)
	}

	return nil
}
