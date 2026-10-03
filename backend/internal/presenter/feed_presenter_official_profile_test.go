package presenter

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/dflh-saf/backend/internal/model"
)

// Every feed post payload apps and the web read carries officialProfile,
// including false, so clients can rely on the key.
func TestFeedPayloadsExposeOfficialProfile(t *testing.T) {
	p := NewFeedPresenter()
	official := &model.NoticeDetail{SEQ: 1, RegName: "대일외고장학회", OfficialProfile: true, ContentFormat: "MARKDOWN"}
	personal := &model.NoticeDetail{SEQ: 2, RegName: "홍길동", ContentFormat: "MARKDOWN"}
	cases := []struct {
		payload interface{}
		want    string
	}{
		{p.FormatNoticeDetail(official), `"officialProfile":true`},
		{p.FormatNoticeDetailForAdmin(personal), `"officialProfile":false`},
		{model.FeedItem{Type: "notice", NoticeItem: &model.NoticeItem{SEQ: 3, OfficialProfile: true}}, `"officialProfile":true`},
		{model.AdminNoticeRow{SEQ: 4, RegName: "홍길동"}, `"officialProfile":false`},
	}
	for _, tc := range cases {
		body, err := json.Marshal(tc.payload)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(body), tc.want) {
			t.Errorf("payload %s is missing %s", body, tc.want)
		}
	}
}
