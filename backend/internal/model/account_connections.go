package model

type AccountConnections struct {
	Providers   []SocialProvider `json:"providers"`
	HasPassword bool             `json:"hasPassword"`
}

func (c AccountConnections) HasProvider(target SocialProvider) bool {
	for _, provider := range c.Providers {
		if provider == target {
			return true
		}
	}
	return false
}

func (c AccountConnections) HasAlternativeTo(target SocialProvider) bool {
	if c.HasPassword {
		return true
	}
	for _, provider := range c.Providers {
		if provider != target {
			return true
		}
	}
	return false
}

type SocialDisconnectStatus string

const (
	SocialDisconnectCompleted    SocialDisconnectStatus = "disconnected"
	SocialDisconnectNotConnected SocialDisconnectStatus = "notConnected"
)

type SocialDisconnectResult struct {
	Status      SocialDisconnectStatus `json:"status"`
	Connections AccountConnections     `json:"connections"`
}
