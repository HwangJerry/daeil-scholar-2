-- Migration 004: Add Markdown content columns to WEO_BOARDBBS
-- Target: MariaDB 10.1.38

-- CONTENTS_MD stores the original Markdown text for editing
-- CONTENT_FORMAT distinguishes new Markdown posts from legacy raw-HTML posts
ALTER TABLE WEO_BOARDBBS
  ADD COLUMN CONTENTS_MD MEDIUMTEXT NULL COMMENT '원본 Markdown 텍스트 (편집용)' AFTER CONTENTS,
  ADD COLUMN CONTENT_FORMAT ENUM('LEGACY','MARKDOWN') DEFAULT 'LEGACY' COMMENT '콘텐츠 포맷' AFTER CONTENTS_MD;
