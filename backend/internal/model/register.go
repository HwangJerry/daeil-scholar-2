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
	Nick    string `json:"nick"`
	Dept    string `json:"dept"`
	JobCat  *int   `json:"jobCat"`
	BizName string `json:"bizName"`
	BizDesc string `json:"bizDesc"`
	BizAddr string `json:"bizAddr"`
}
