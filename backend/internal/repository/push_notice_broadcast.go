// push_notice_broadcast.go — Recipient lookup for the notice (새 소식) push broadcast.
package repository

// ListNoticeRecipientSeqs returns one page of members who have at least one
// active device and have not opted out of notice pushes, ordered by USR_SEQ and
// keyset-paginated on afterUserSeq.
//
// Only the member sequence is selected: the delivery worker refetches that
// member's devices anyway, so returning device rows here would read — and
// discard — several times the data on every page.
//
// A page is therefore a page of members, and a member can never straddle a page
// boundary, so paging can neither enqueue the same recipient twice nor skip
// one. MariaDB 10.1 has no window functions, but a keyset walk needs none.
func (r *PushRepository) ListNoticeRecipientSeqs(afterUserSeq int, limit int) ([]int, error) {
	if limit <= 0 {
		return nil, nil
	}
	seqs := make([]int, 0, limit)
	err := r.db.Select(&seqs, `
		SELECT DISTINCT d.USR_SEQ
		FROM ALUMNI_MOBILE_DEVICE_TOKEN d
		LEFT JOIN ALUMNI_PUSH_PREFERENCE p ON p.USR_SEQ = d.USR_SEQ
		WHERE d.STATUS = 'ACTIVE'
		  AND COALESCE(p.NOTICE_ENABLED, 'Y') = 'Y'
		  AND d.USR_SEQ > ?
		ORDER BY d.USR_SEQ ASC
		LIMIT ?
	`, afterUserSeq, limit)
	if err != nil {
		return nil, err
	}
	return seqs, nil
}

// NoticeStillPublished reports whether the notice is still visible. A broadcast
// can outlive the notice it announces — an administrator may delete it while
// the fan-out is still draining — and a push must never deep-link to a notice
// that is gone.
func (r *PushRepository) NoticeStillPublished(noticeSeq int) (bool, error) {
	var count int
	err := r.db.Get(&count, `SELECT COUNT(*) FROM WEO_BOARDBBS
		WHERE SEQ = ? AND GATE = 'NOTICE' AND OPEN_YN = 'Y'`, noticeSeq)
	return count > 0, err
}
