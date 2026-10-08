// message_preferences.go — Account-wide direct-message receiving preference.
package model

type MessagePreferences struct {
	MessageAllowed bool `json:"messageAllowed"`
}
