-- 013_add_performance_indexes.sql — Performance indexes for feed and alumni queries

-- Feed: 공지사항 목록 조회 (GATE, OPEN_YN 필터링 + SEQ cursor 기반 pagination)
CREATE INDEX IDX_BBS_FEED ON WEO_BOARDBBS (GATE, OPEN_YN, SEQ);

-- Feed: 좋아요 수 조회
CREATE INDEX IDX_BBS_LIKE ON WEO_BOARDLIKE (BBS_SEQ, OPEN_YN);

-- Feed: 댓글 수 조회
CREATE INDEX IDX_BCOM_JOIN ON WEO_BOARDCOMAND (JOIN_SEQ, BC_TYPE, OPEN_YN);

-- Alumni: 기수(FN) 필터 — FUNDAMENTAL_MEMBER
CREATE INDEX IDX_FM_FN ON FUNDAMENTAL_MEMBER (FM_FN);

-- Alumni: 탈퇴 여부 필터 — 전체 동문 쿼리에 공통 사용
CREATE INDEX IDX_USR_STATUS ON WEO_MEMBER (USR_STATUS);

-- Alumni: 주간 신규 동문 카운트
CREATE INDEX IDX_USR_REG_DATE ON WEO_MEMBER (REG_DATE);
