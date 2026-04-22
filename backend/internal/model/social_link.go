// social_link.go — Data held server-side during the social account linking flow
package model

// SocialLinkData holds social provider user info stored server-side during the account linking flow.
type SocialLinkData struct {
	Provider        string // "KT" (Kakao), "NV" (Naver), "FB" (Facebook)
	SocialID        string
	Email           string
	Nickname        string
	ProfileImageURL string // empty when the provider did not supply an image (e.g. consent declined)
	AccessToken     string
}

// KakaoLinkData is kept as an alias for backward compatibility with cached data.
type KakaoLinkData = SocialLinkData
