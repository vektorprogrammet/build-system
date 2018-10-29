package messenger

import (
	"net/http"
	"bytes"
	"encoding/json"
)

type Slack struct {
	Endpoint  string `json:"endpoint"`
	Channel   string `json:"channel"`
	Username  string `json:"username"`
	IconEmoji string `json:"icon_emoji"`
}

type slackMessage struct {
	Channel   string `json:"channel"`
	Text      string `json:"text"`
	Username  string `json:"username"`
	IconEmoji string `json:"icon_emoji"`
}

func (s Slack) Send(message string) {
	jsonData, _ := json.Marshal(slackMessage{
		Channel:   s.Channel,
		Text:      message,
		Username:  s.Username,
		IconEmoji: s.IconEmoji,
	})
	http.Post(s.Endpoint, "application/json", bytes.NewBuffer(jsonData))
}

func NewSlack(endpoint, channel, username, iconEmoji string) Slack {
	return Slack{
		Channel:   channel,
		Endpoint:  endpoint,
		Username:  username,
		IconEmoji: iconEmoji,
	}
}
