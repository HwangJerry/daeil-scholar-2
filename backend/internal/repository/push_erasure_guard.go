// push_erasure_guard.go — Drop queued message previews after withdrawal or content removal.
package repository

func (r *PushRepository) MessageStillAvailable(sender, recipient int, messageID int64) (bool, error) {
	var n int
	err := r.db.Get(&n, `SELECT COUNT(*) FROM ALUMNI_MESSAGE m
        JOIN WEO_MEMBER s ON s.USR_SEQ=m.AM_SENDER_SEQ AND s.USR_STATUS<>'AAA'
        JOIN WEO_MEMBER r ON r.USR_SEQ=m.AM_RECVR_SEQ AND r.USR_STATUS<>'AAA'
        WHERE m.AM_SEQ=? AND m.AM_SENDER_SEQ=? AND m.AM_RECVR_SEQ=?`, messageID, sender, recipient)
	return n == 1, err
}
