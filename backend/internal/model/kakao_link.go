package model

// KakaoLinkData holds Kakao user info stored server-side during the account linking flow.
type KakaoLinkData struct {
	KakaoID     string
	Email       string
	Nickname    string
	AccessToken string
}
