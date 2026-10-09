package repository

import "testing"

func TestSignupPhotoCandidateRetainsEncodedNamesAndScopesMalformedURLs(t *testing.T) {
	local := "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa.jpg"
	filter := signupPhotoReferenceCandidate(local)
	for _, raw := range []string{local, "https://www.app.example.test" + local + "?x=1#p", "/uploads/profile/%61aaaaaaaaaaaaaaaaaaaaaaa.jpg", "https:/uploads/profile/%61aaaaaaaaaaaaaaaaaaaaaaa.jpg?bad=%zz", "/uploads/profile/aaaaaaaaaaaaaaaaaaaaaaaa%2ejpg", "/uploads/profile/%2561aaaaaaaaaaaaaaaaaaaaaaa.jpg"} {
		if !filter(raw) {
			t.Errorf("candidate was missed: %s", raw)
		}
	}
	for _, raw := range []string{"https:/uploads/profile/unrelated.jpg", "/files/%zz", "https://other.test/picture.jpg"} {
		if filter(raw) {
			t.Errorf("unrelated malformed URL was treated as candidate: %s", raw)
		}
	}
}
