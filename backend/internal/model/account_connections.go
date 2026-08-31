package model

type AccountConnections struct {
	Providers   []string `json:"providers"`
	HasPassword bool     `json:"hasPassword"`
}

type SocialDisconnectStatus string

const (
	SocialDisconnectStatusDisconnected SocialDisconnectStatus = "disconnected"
	SocialDisconnectStatusNotConnected SocialDisconnectStatus = "notConnected"
)

type SocialDisconnectResult struct {
	Status      SocialDisconnectStatus `json:"status"`
	Connections AccountConnections     `json:"connections"`
}
