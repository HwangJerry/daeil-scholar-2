-- Migration 026: WEO_FILES.REG_DATE 컬럼 추가 (레거시 스키마 보정)
-- Target: MariaDB 10.1.38 (`ADD COLUMN IF NOT EXISTS` 10.0.2+ 지원으로 멱등)
--
-- 사유:
--   레거시 PHP 시절 생성된 WEO_FILES 테이블에 REG_DATE 컬럼이 없어,
--   백엔드의 INSERT (file_repo.go:InsertFile) 가 ERROR 1054로 거절되어
--   모든 첨부/이미지/프로필 사진 업로드가 500 으로 실패하던 문제를 해결.

ALTER TABLE WEO_FILES
  ADD COLUMN IF NOT EXISTS REG_DATE DATETIME NULL COMMENT '파일 등록 시각' AFTER OPEN_YN;
