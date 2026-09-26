// admin_member_profile.go — Admin member search filter and profile edit request types
package model

// AdminMemberFilter narrows the admin member list. Empty fields do not filter.
// RegFrom and RegTo are inclusive YYYY-MM-DD join dates.
type AdminMemberFilter struct {
	Query   string
	FN      string
	Dept    string
	Status  string
	RegFrom string
	RegTo   string
}

// AdminMemberProfileUpdate is the set of member fields an administrator may
// correct. Empty USRPhone, USRFN and USRDept keep the stored value; USREmail is
// written as given so a contact email can be cleared.
type AdminMemberProfileUpdate struct {
	USRName  string `json:"usrName"`
	USRPhone string `json:"usrPhone"`
	USREmail string `json:"usrEmail"`
	USRFN    string `json:"usrFn"`
	USRDept  string `json:"usrDept"`
}
