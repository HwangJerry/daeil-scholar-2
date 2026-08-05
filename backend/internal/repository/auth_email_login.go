// auth_email_login.go — Canonical email/password member lookup.
package repository

import "github.com/dflh-saf/backend/internal/model"

func (r *AuthRepository) FindMemberByEmailAndPwdAny(email, hashedPwd string) (*model.User, error) {
	var users []model.User
	if err := r.DB.Select(&users, `
		SELECT USR_SEQ, USR_ID, USR_NAME, USR_STATUS, USR_PHONE, USR_FN, USR_EMAIL, USR_NICK, USR_PHOTO
		FROM WEO_MEMBER
		WHERE LOWER(TRIM(USR_EMAIL)) = LOWER(TRIM(?)) AND USR_PWD = ?
		LIMIT 2
	`, email, hashedPwd); err != nil {
		return nil, err
	}
	if len(users) != 1 {
		return nil, nil
	}
	return &users[0], nil
}
