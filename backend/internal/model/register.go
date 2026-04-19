// register.go — Registration request DTO for ID/password signup.
package model

// RegisterRequest is the request body for POST /api/auth/register.
type RegisterRequest struct {
	UsrID    string `json:"usrId"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	FN       string `json:"fn"`
	Email    string `json:"email"`
	// Optional profile fields collected at signup time.
	Nick           string   `json:"nick"`
	FmDept         string   `json:"fmDept"`
	JobCat         *int     `json:"jobCat"`
	BizName        string   `json:"bizName"`
	BizDesc        string   `json:"bizDesc"`
	BizAddr        string   `json:"bizAddr"`
	Position       string   `json:"position"`
	Tags           []string `json:"tags"`
	USRPhonePublic string   `json:"usrPhonePublic"` // 'Y' | 'N'
	USREmailPublic string   `json:"usrEmailPublic"` // 'Y' | 'N'
}
