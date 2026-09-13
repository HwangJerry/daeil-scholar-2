// account_erasure_steps.go — Code-owned erasure rules shared by the worker and the operator preview.
package repository

// erasureStep is one DELETE predicate. Tables and predicates are constants
// owned by this file; only member identifiers are bound as arguments.
type erasureStep struct {
	table string
	where string
	args  []interface{}
}

// anonymizedColumn is one WEO_BOARDBBS column overwritten instead of deleted.
type anonymizedColumn struct {
	column string
	value  interface{}
}

const erasedPostText = "탈퇴한 회원의 삭제된 게시글입니다."

var erasureMemberTables = []string{
	"ALUMNI_PUSH_OUTBOX", "ALUMNI_MOBILE_DEVICE_TOKEN", "ALUMNI_PUSH_DEVICE", "ALUMNI_PUSH_PREFERENCE",
	"ALUMNI_MOBILE_REFRESH_TOKEN", "ALUMNI_NOTIFICATION", "ALUMNI_PASSWORD_RESET", "USER_SESSION",
	"WEO_MEMBER_LOG", "WEO_MEMBER_PUSH", "WEO_MEMBER_OUT", "WEO_PUSH_NOTI", "WEO_SMS", "WEO_APP_PUSH",
	"ALUMNI_USER_TAG", "ALUMNI_ADMIN_ROLE", "ALUMNI_VERIFICATION", "FUNDAMENTAL_MEMBER",
	"WEO_BOARDLIKE", "WEO_BOARDCOMAND", "WEO_AD_COMMENT", "WEO_AD_LIKE", "WEO_AD_LOG",
}

var boardPersonalColumns = []string{"SUBJECT", "CONTENTS", "CONTENTS_MD", "SUMMARY", "THUMBNAIL_URL", "FILES", "RE_FILES", "USR_NAME", "USR_ID", "EMAIL", "PHONE", "IP", "BBS_IP", "PASSWORD", "REG_ID", "REG_NAME", "REG_EMAIL", "REG_TEL", "REG_PWD", "REG_IPADDR"}

// boardAnonymizedColumns keeps other authors' conversation by retaining an
// anonymous, empty parent post. USR_SEQ=0 detaches the withdrawn author.
func boardAnonymizedColumns(s erasureSchema) []anonymizedColumn {
	columns := []anonymizedColumn{{"USR_SEQ", 0}}
	for _, column := range boardPersonalColumns {
		if s.has("WEO_BOARDBBS", column) {
			value := ""
			if column == "SUBJECT" || column == "CONTENTS" {
				value = erasedPostText
			}
			columns = append(columns, anonymizedColumn{column, value})
		}
	}
	return columns
}

// accountReferenceSteps lists, in execution order, every member-linked deletion
// performed after donations, files and posts. Order matters: subqueries read
// source rows that later steps delete.
func accountReferenceSteps(s erasureSchema, user int, email string) []erasureStep {
	steps := []erasureStep{}
	if s["ALUMNI_MESSAGE"] != nil {
		steps = append(steps,
			erasureStep{"ALUMNI_PUSH_OUTBOX", "EVENT_TYPE='message' AND EVENT_ID IN (SELECT AM_SEQ FROM ALUMNI_MESSAGE WHERE AM_SENDER_SEQ=? OR AM_RECVR_SEQ=?)", []interface{}{user, user}},
			erasureStep{"ALUMNI_NOTIFICATION", "AN_TYPE='message' AND AN_REF_SEQ IN (SELECT AM_SEQ FROM ALUMNI_MESSAGE WHERE AM_SENDER_SEQ=? OR AM_RECVR_SEQ=?)", []interface{}{user, user}},
		)
	}
	for _, table := range erasureMemberTables {
		steps = append(steps, erasureStep{table, "USR_SEQ=?", []interface{}{user}})
	}
	// Current banner statistics contain no member identifier. Older deployments
	// may have a member-linked variant; erase only that explicit variant.
	if s.has("WEO_BANNER_AD_LOG", "USR_SEQ") {
		steps = append(steps, erasureStep{"WEO_BANNER_AD_LOG", "USR_SEQ=?", []interface{}{user}})
	}
	steps = append(steps,
		erasureStep{"ALUMNI_MESSAGE_REPORT", "REPORTER_SEQ=? OR REPORTED_SEQ=?", []interface{}{user, user}},
		erasureStep{"ALUMNI_MEMBER_BLOCK", "BLOCKER_USR_SEQ=? OR BLOCKED_USR_SEQ=?", []interface{}{user, user}},
		erasureStep{"ALUMNI_MESSAGE", "AM_SENDER_SEQ=? OR AM_RECVR_SEQ=?", []interface{}{user, user}},
		erasureStep{"ALUMNI_MOBILE_APP_EVENT", "USER_ID=?", []interface{}{user}},
		erasureStep{"WEO_VISIT_DAILY", "VD_USR_SEQ=?", []interface{}{user}},
		erasureStep{"AUTH_EMAIL_VERIFICATION", "NORMALIZED_EMAIL=? AND ?<>''", []interface{}{email, email}},
	)
	if s["AUTH_IDENTITY"] != nil {
		steps = append(steps, erasureStep{"AUTH_EMAIL_VERIFICATION", "NORMALIZED_EMAIL IN (SELECT NORMALIZED_EMAIL FROM AUTH_IDENTITY WHERE ACCOUNT_ID=?)", []interface{}{user}})
		if s["AUTH_SIGNUP_CONTINUATION"] != nil {
			subjectMatch := "EXISTS (SELECT 1 FROM AUTH_IDENTITY i WHERE i.ACCOUNT_ID=? AND i.PROVIDER=AUTH_SIGNUP_CONTINUATION.PROVIDER AND i.SUBJECT_KEY=AUTH_SIGNUP_CONTINUATION.SUBJECT_KEY)"
			steps = append(steps,
				erasureStep{"AUTH_PROVIDER_CREDENTIAL", "CONTINUATION_TOKEN_HASH IN (SELECT TOKEN_HASH FROM AUTH_SIGNUP_CONTINUATION WHERE " + subjectMatch + ")", []interface{}{user}},
				erasureStep{"AUTH_SIGNUP_CONTINUATION", subjectMatch, []interface{}{user}},
			)
		}
		for _, table := range []string{"AUTH_PROVIDER_REVOKE_OUTBOX", "AUTH_SESSION_FAMILY", "AUTH_PASSWORD_CREDENTIAL", "AUTH_PROVIDER_CREDENTIAL"} {
			steps = append(steps, erasureStep{table, "IDENTITY_ID IN (SELECT IDENTITY_ID FROM AUTH_IDENTITY WHERE ACCOUNT_ID=?)", []interface{}{user}})
		}
	}
	for _, table := range []string{"AUTH_PHONE_CLAIM", "AUTH_CONSENT", "AUTH_IDENTITY", "AUTH_ACCOUNT_STATE"} {
		steps = append(steps, erasureStep{table, "ACCOUNT_ID=?", []interface{}{user}})
	}
	for _, table := range []string{"WEO_MEMBER_SOCIAL", "ALUMNI_SOCIAL_CREDENTIAL", "ALUMNI_UPLOAD_OWNER", "ALUMNI_PROFILE_FILE_HISTORY"} {
		steps = append(steps, erasureStep{table, "USR_SEQ=?", []interface{}{user}})
	}
	return steps
}
