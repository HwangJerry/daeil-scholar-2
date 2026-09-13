package repository

func (r *AuthRepository) AccountDeletionKakaoSubject(user int) (string, error) {
	var subject string
	err := r.DB.Get(&subject, `SELECT s.NMS_ID FROM WEO_MEMBER_SOCIAL s JOIN WEO_MEMBER m ON m.USR_SEQ=s.USR_SEQ WHERE s.USR_SEQ=? AND s.NMS_GATE='KT' AND m.USR_STATUS='AAA'`, user)
	return subject, err
}
