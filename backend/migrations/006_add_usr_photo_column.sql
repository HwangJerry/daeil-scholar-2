-- 006_add_usr_photo_column.sql
-- WEO_MEMBER 테이블에 USR_PHOTO 컬럼 추가
-- 로그인 시 auth_repo.go 쿼리가 USR_PHOTO를 SELECT하지만 컬럼이 없어 500 에러 발생
-- profile_repo.go, admin_member_repo.go 등도 동일 컬럼 참조

ALTER TABLE WEO_MEMBER ADD COLUMN USR_PHOTO VARCHAR(500) DEFAULT NULL;
