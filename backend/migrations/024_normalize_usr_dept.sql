-- 024_normalize_usr_dept.sql
-- Normalize WEO_MEMBER.USR_DEPT into the canonical 7-value enum:
--   프랑스어, 독일어, 일본어, 중국어, 스페인어, 러시아어, 영어
-- Source-of-truth: backend/internal/model/departments.go
-- Mapping derived from production diagnostic SELECT (2026-04-30).
-- Unmappable / non-language values (e.g. '운영팀', '국제어과', '') become NULL.

UPDATE WEO_MEMBER SET USR_DEPT = '프랑스어'
  WHERE USR_DEPT IN ('불어과', '불', '프', '프랑스어과');

UPDATE WEO_MEMBER SET USR_DEPT = '독일어'
  WHERE USR_DEPT IN ('독일어과', '독');

UPDATE WEO_MEMBER SET USR_DEPT = '일본어'
  WHERE USR_DEPT IN ('일어과', '일', '일본어과');

UPDATE WEO_MEMBER SET USR_DEPT = '중국어'
  WHERE USR_DEPT IN ('중국어과', '중');

UPDATE WEO_MEMBER SET USR_DEPT = '스페인어'
  WHERE USR_DEPT IN ('서어과', '스페인어과', '서', '스');

UPDATE WEO_MEMBER SET USR_DEPT = '러시아어'
  WHERE USR_DEPT IN ('러시아어과', '러');

UPDATE WEO_MEMBER SET USR_DEPT = '영어'
  WHERE USR_DEPT IN ('영어과', '영');

-- Catch-all: anything not matching the canonical 7 values becomes NULL.
-- This handles empty strings, '운영팀', '국제어과', and any future stray values.
UPDATE WEO_MEMBER
  SET USR_DEPT = NULL
  WHERE USR_DEPT IS NOT NULL
    AND USR_DEPT NOT IN ('프랑스어','독일어','일본어','중국어','스페인어','러시아어','영어');
