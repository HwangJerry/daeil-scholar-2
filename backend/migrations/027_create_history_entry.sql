-- Migration 027: Create history entry table and seed existing data
-- Target: MariaDB 10.1.38

CREATE TABLE HISTORY_ENTRY (
    HE_SEQ        INT AUTO_INCREMENT PRIMARY KEY,
    HE_EVENT_DATE DATE         NOT NULL,
    HE_TEXT       VARCHAR(500) NOT NULL,
    HE_SORT_ORDER SMALLINT     NOT NULL DEFAULT 0,
    REG_DATE      DATETIME     DEFAULT CURRENT_TIMESTAMP,
    MOD_DATE      DATETIME     DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX IDX_DATE (HE_EVENT_DATE DESC, HE_SORT_ORDER ASC)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed with existing hardcoded data (ordered oldest→newest so AUTO_INCREMENT
-- matches chronological order; display order is controlled by HE_EVENT_DATE DESC)

INSERT INTO HISTORY_ENTRY (HE_EVENT_DATE, HE_TEXT, HE_SORT_ORDER) VALUES
-- 2016
('2016-12-01', '장학재단 설립 추진을 위한 역대 동문회장단 미팅', 0),
('2016-12-09', '대일외고 장학재단 설립 추진위원회 구성 (위원장: 엄은숙, 부위원장: 이종민)', 1),
-- 2017
('2017-01-12', '대일외고 장학재단 준비위원회 설명회', 0),
('2017-07-14', 'Dream for High School Students (미주 대일외고 장학회) 설립', 1),
('2017-12-04', '대일외고 장학회 창립총회 (회장: 허재혁)', 2),
-- 2018
('2018-12-29', '대일외고 장학회 현판식 및 비전선포식', 0),
-- 2019
('2019-08-26', '비영리민간단체 등록 (서울특별시, 등록번호: 2360)', 0),
('2019-12-31', '기부금대상민간단체 지정 (기획재정부 공고 제2019-219호)', 1),
-- 2020
('2020-03-21', '대일외고 장학회 제2차 현판식', 0),
-- 2022
('2022-07-07', '서울대 최종학 교수 초청 특강', 0),
('2022-08-22', '후원인의 밤', 1),
('2022-09-10', '대일외고 장학회 남춘천CC 골프', 2),
('2022-11-05', '대일외고 장학회 제3차 현판식', 3),
-- 2023
('2023-10-21', '대일외고 장학회 제4차 현판식', 0),
-- 2024
('2024-01-04', '제1회 그랜드마스터클래스 — 우리 아이들에게 펼쳐질 미래 (김승주)', 0),
('2024-07-05', '제2회 그랜드마스터클래스 — 인스타그램, 트렌드를 보는 창 (정다정)', 1),
('2024-08-21', '서울대 최종학 교수 초청 특강', 2),
('2024-10-26', '대일외고 장학회 제5차 현판식', 3),
-- 2025
('2025-01-09', '제3차 조찬 세미나 — 실패 없는 창업 성공법칙 (임상진)', 0),
('2025-02-23', '제1회 재능기부콘서트 — 취업이냐, 창업이냐, 그것이 문제로다 (이준용)', 1),
('2025-05-28', '제2회 재능기부콘서트 — 창업 시 필요한 법률 지식 (허정무)', 2),
('2025-06-26', '후원인의 밤', 3),
('2025-07-28', '제3회 재능기부콘서트 — 절세법과 자산 관리의 노하우 (정용호)', 4),
('2025-09-02', '제4회 그랜드마스터클래스 — 격변의 시대, 자본시장을 읽다 (신창훈)', 5),
('2025-10-23', '제4회 재능기부콘서트 — 안녕하세요? 미래에서 왔습니다! (유효현)', 6),
('2025-10-24', '대일외고 장학회 제6차 현판식', 7);
