// auth_social_phone_match_handler.go — Phone-based existing-member lookup for signup form merge mode
package handler

import (
	"net/http"
)

type phoneMatchProfile struct {
	Name           string   `json:"name"`
	Email          string   `json:"email"`
	FN             string   `json:"fn"`
	FmDept         string   `json:"fmDept"`
	JobCat         *int     `json:"jobCat"`
	BizName        string   `json:"bizName"`
	BizDesc        string   `json:"bizDesc"`
	BizAddr        string   `json:"bizAddr"`
	Position       string   `json:"position"`
	Tags           []string `json:"tags"`
	USRPhonePublic string   `json:"usrPhonePublic"`
	USREmailPublic string   `json:"usrEmailPublic"`
}

type phoneMatchResponse struct {
	Matched bool               `json:"matched"`
	Profile *phoneMatchProfile `json:"profile,omitempty"`
}

// SocialLinkPhoneMatch looks up an existing member by phone, returning a prefill payload
// when matched. Requires a valid social-link token so only Kakao-authenticated callers can query.
// The cache is NOT consumed here.
func (h *AuthHandler) SocialLinkPhoneMatch(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	phone := r.URL.Query().Get("phone")
	if token == "" {
		respondError(w, http.StatusBadRequest, "MISSING_TOKEN", "token 파라미터가 필요합니다")
		return
	}
	if phone == "" {
		respondError(w, http.StatusBadRequest, "MISSING_PHONE", "phone 파라미터가 필요합니다")
		return
	}
	if _, found := h.cache.Get("social_link:" + token); !found {
		respondError(w, http.StatusNotFound, "TOKEN_NOT_FOUND", "유효한 소셜 링크 토큰이 아닙니다")
		return
	}
	existing, err := h.memberSvc.FindMemberByPhone(phone)
	if err != nil {
		h.logger.Error().Err(err).Str("phone", phone).Msg("social phone-match: lookup failed")
		respondError(w, http.StatusInternalServerError, "LOOKUP_FAILED", "회원 조회에 실패했습니다")
		return
	}
	if existing == nil {
		respondJSON(w, http.StatusOK, phoneMatchResponse{Matched: false})
		return
	}
	profile, err := h.profileSvc.GetProfile(existing.USRSeq)
	if err != nil {
		h.logger.Error().Err(err).Int("usrSeq", existing.USRSeq).Msg("social phone-match: profile fetch failed")
		respondError(w, http.StatusInternalServerError, "PROFILE_FAILED", "회원 프로필 조회에 실패했습니다")
		return
	}
	var jobCat *int
	if profile.JobCat > 0 {
		jc := profile.JobCat
		jobCat = &jc
	}
	respondJSON(w, http.StatusOK, phoneMatchResponse{
		Matched: true,
		Profile: &phoneMatchProfile{
			Name:           profile.USRName,
			Email:          profile.USREmail,
			FN:             profile.USRFN,
			FmDept:         profile.FmDept,
			JobCat:         jobCat,
			BizName:        profile.BizName,
			BizDesc:        profile.BizDesc,
			BizAddr:        profile.BizAddr,
			Position:       profile.Position,
			Tags:           profile.Tags,
			USRPhonePublic: profile.USRPhonePublic,
			USREmailPublic: profile.USREmailPublic,
		},
	})
}
