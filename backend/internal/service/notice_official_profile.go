// notice_official_profile.go — Resolves the byline (official profile flag + REG_NAME) an admin feed post is stored with
package service

// OfficialProfileName is the byline a post published under the foundation's
// official profile shows in place of the operator's name.
const OfficialProfileName = "대일외고장학회"

// noticeByline is what a post stores as its author: OFFICIAL_PROFILE_YN and
// REG_NAME. On update, empty fields keep the stored values.
type noticeByline struct {
	officialProfileYN string
	regName           string
}

// newNoticeByline resolves a new post's byline. A missing choice (nil) means
// the official profile, which is the default for new posts; otherwise the
// post carries the writing operator's name.
func newNoticeByline(officialProfile *bool, adminName string) noticeByline {
	if officialProfile == nil || *officialProfile {
		return noticeByline{officialProfileYN: "Y", regName: OfficialProfileName}
	}
	return noticeByline{officialProfileYN: "N", regName: adminName}
}

// editedNoticeByline resolves an edited post's byline. A missing choice keeps
// it unchanged. Turning the official profile off restores the real name of the
// post's USR_SEQ author, falling back to the editing operator's name when that
// member is gone; a post that was already personal keeps its stored name.
func (s *AdminNoticeService) editedNoticeByline(seq int, officialProfile *bool, editorName string) (noticeByline, error) {
	if officialProfile == nil {
		return noticeByline{}, nil
	}
	if *officialProfile {
		return noticeByline{officialProfileYN: "Y", regName: OfficialProfileName}, nil
	}
	stored, err := s.repo.GetNoticeAuthorProfile(seq)
	if err != nil {
		return noticeByline{}, err
	}
	wasOfficial := stored != nil && stored.OfficialProfileYN == "Y"
	if !wasOfficial {
		return noticeByline{officialProfileYN: "N"}, nil
	}
	if stored.AuthorName != "" {
		return noticeByline{officialProfileYN: "N", regName: stored.AuthorName}, nil
	}
	return noticeByline{officialProfileYN: "N", regName: editorName}, nil
}
