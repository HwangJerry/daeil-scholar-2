# Alumni full profile (V10)

`GET /api/alumni/{userSeq}` now returns the complete approved member profile.
Existing authentication, approval eligibility, status filters and block state
remain in place. Search/list responses are unchanged.

| JSON field | Profile display |
| --- | --- |
| name | 이름 |
| graduationYear (nullable integer) | 졸업연도 |
| cohort | 기수 |
| department | 학과 |
| phone / phonePublic (boolean) | 휴대폰번호 / 공개 여부 |
| email / emailPublic (boolean) | 이메일 / 공개 여부 |
| jobCategory | 업종 |
| bizName | 소속 |
| bizAddr | 근무지 |
| jobRole | 직책 |
| bizDesc | 소개글 |
| tags (ordered string array) | 태그 |
| bizCardUrl (nullable string) | 명함 이미지 |

A contact value is omitted unless its own stored visibility flag is exactly Y.
The boolean is returned even when the public value is missing, so clients can
show `정보없음` for public empty data and `비공개` for private data independently.
Null/unknown flags are private. Hidden contact values are not serialized.
Academic fields come from ALUMNI_VERIFICATION; business fields from WEO_MEMBER;
tags from ALUMNI_USER_TAG ordered by AUT_INDX then AUT_SEQ. Tag read errors fail
the request instead of silently presenting saved tags as missing.

No schema migration is needed. Deploy this additive backend response before the
mobile versions: older clients ignore new fields; new clients fail closed when
visibility booleans are absent in an older server response.

Validation: alumni service, handler and repository tests; disclosure cases include
public/private combinations, public empty data, unknown visibility, complete
business fields and ordered tags. This feature change does not deploy production.
