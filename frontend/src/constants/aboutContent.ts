// aboutContent — Static copy/data extracted from v1 _TPL_scholarship_*.htm for the v2 about pages

export const ABOUT_INTRO = [
  '대일외고 장학회는 2017년 12월 4일 정식으로 창립된 후, 모교 후배들에게 안정적이고 체계적인 장학금을 전달해 왔습니다.',
  '미주·호주를 비롯한 전 세계 동문, 기수모임, 동호회의 마음이 모여 운영되며, 2019년 비영리민간단체 등록과 기부금대상민간단체 지정을 거치며 신뢰의 기반을 다져왔습니다.',
] as const;

export interface AboutLink {
  to: string;
  title: string;
  description: string;
}

export const ABOUT_LINKS: readonly AboutLink[] = [
  { to: '/greetings', title: '인사말', description: '장학회장의 인사와 운영 철학을 전합니다.' },
  { to: '/vision', title: '비전', description: '미션과 비전, 핵심가치를 소개합니다.' },
  { to: '/history', title: '연혁', description: '장학회가 걸어온 길을 연도별로 정리했습니다.' },
  { to: '/organization', title: '조직도', description: '장학회를 이끄는 임원과 사무국을 소개합니다.' },
  { to: '/business', title: '사업소개', description: '후배들을 위한 주요 장학사업을 소개합니다.' },
];

export const FOOTER_INFO_LINKS: readonly { to: string; label: string }[] = [
  { to: '/about', label: '장학회 소개' },
  { to: '/greetings', label: '인사말' },
  { to: '/vision', label: '비전' },
  { to: '/history', label: '연혁' },
  { to: '/organization', label: '조직도' },
  { to: '/business', label: '사업소개' },
];

export const GREETINGS = {
  salutation: '안녕하세요. 대일외고 장학회장 4기 졸업생 엄은숙입니다.',
  paragraphs: [
    '대일외고 졸업생들은 모교 후배들을 위해 다양한 형태의 내리사랑을 실천하고 있으며, 2017년 12월 4일 장학회를 정식으로 창립하여 체계적으로 장학금을 전달하고 있습니다.',
    '우리 장학회의 기금은 많은 동문들이 직접 동참해 주신 덕분이며, 멀리 미주와 호주, 그리고 각 개인뿐 아니라 기수모임·동호회에서까지 기금을 보내주셨습니다.',
    '저희 장학회 임원들은 이러한 동문들의 뜻을 담아 보다 안정적으로 기금을 운용하고, 적재적소에 장학금을 전달하여 대일외고인들이 훌륭한 인재로 성장하는 데 일조할 수 있도록 최선을 다하겠습니다.',
  ],
  closing: '감사합니다.',
  signature: '장학회장 엄은숙',
} as const;

export const VISION_MISSION = {
  label: 'Mission',
  body: '후배들의 아름다운 꿈과 높은 이상을 실현하게 한다.',
} as const;

export const VISION_VISION = {
  label: 'Vision',
  body: '지속적이고 안정적인 후원 활동을 하는 장학회',
  goal: '[Goal] 2021년 장학재단 설립',
} as const;

export interface VisionCoreValue {
  title: string;
  bullets: string[];
}

export const VISION_CORE_VALUES: readonly VisionCoreValue[] = [
  {
    title: '인재 육성',
    bullets: [
      '안정적인 장학금 지원역량 확보 (모금)',
      '장학금 지원의 효과성 제고',
    ],
  },
  {
    title: '학교 위상 제고',
    bullets: [
      '학교 브랜드 가치 향상',
      '가고 싶은 학교 만들기',
    ],
  },
  {
    title: '운영 건전성 최우선',
    bullets: [
      '장학기금 운영 투명성 유지',
      '학교·동문과의 활발한 소통',
    ],
  },
];

export interface HistoryItem {
  date: string;
  text: string;
}

export interface HistoryYear {
  year: number;
  items: HistoryItem[];
}

export const HISTORY_ENTRIES: readonly HistoryYear[] = [
  {
    year: 2025,
    items: [
      { date: '10.24', text: '대일외고 장학회 제6차 현판식' },
      { date: '10.23', text: '제4회 재능기부콘서트 — 안녕하세요? 미래에서 왔습니다! (유효현)' },
      { date: '09.02', text: '제4회 그랜드마스터클래스 — 격변의 시대, 자본시장을 읽다 (신창훈)' },
      { date: '07.28', text: '제3회 재능기부콘서트 — 절세법과 자산 관리의 노하우 (정용호)' },
      { date: '06.26', text: '후원인의 밤' },
      { date: '05.28', text: '제2회 재능기부콘서트 — 창업 시 필요한 법률 지식 (허정무)' },
      { date: '02.23', text: '제1회 재능기부콘서트 — 취업이냐, 창업이냐, 그것이 문제로다 (이준용)' },
      { date: '01.09', text: '제3차 조찬 세미나 — 실패 없는 창업 성공법칙 (임상진)' },
    ],
  },
  {
    year: 2024,
    items: [
      { date: '10.26', text: '대일외고 장학회 제5차 현판식' },
      { date: '08.21', text: '서울대 최종학 교수 초청 특강' },
      { date: '07.05', text: '제2회 그랜드마스터클래스 — 인스타그램, 트렌드를 보는 창 (정다정)' },
      { date: '01.04', text: '제1회 그랜드마스터클래스 — 우리 아이들에게 펼쳐질 미래 (김승주)' },
    ],
  },
  {
    year: 2023,
    items: [
      { date: '10.21', text: '대일외고 장학회 제4차 현판식' },
    ],
  },
  {
    year: 2022,
    items: [
      { date: '11.05', text: '대일외고 장학회 제3차 현판식' },
      { date: '09.10', text: '대일외고 장학회 남춘천CC 골프' },
      { date: '08.22', text: '후원인의 밤' },
      { date: '07.07', text: '서울대 최종학 교수 초청 특강' },
    ],
  },
  {
    year: 2020,
    items: [
      { date: '03.21', text: '대일외고 장학회 제2차 현판식' },
    ],
  },
  {
    year: 2019,
    items: [
      { date: '12.31', text: '기부금대상민간단체 지정 (기획재정부 공고 제2019-219호)' },
      { date: '08.26', text: '비영리민간단체 등록 (서울특별시, 등록번호: 2360)' },
    ],
  },
  {
    year: 2018,
    items: [
      { date: '12.29', text: '대일외고 장학회 현판식 및 비전선포식' },
    ],
  },
  {
    year: 2017,
    items: [
      { date: '12.04', text: '대일외고 장학회 창립총회 (회장: 허재혁)' },
      { date: '07.14', text: 'Dream for High School Students (미주 대일외고 장학회) 설립' },
      { date: '01.12', text: '대일외고 장학재단 준비위원회 설명회' },
    ],
  },
  {
    year: 2016,
    items: [
      { date: '12.09', text: '대일외고 장학재단 설립 추진위원회 구성 (위원장: 엄은숙, 부위원장: 이종민)' },
      { date: '12.01', text: '장학재단 설립 추진을 위한 역대 동문회장단 미팅' },
    ],
  },
];

export interface OrgPerson {
  name: string;
  cohort: string;
  role?: string;
}

export interface OrgGroup {
  name: string;
  lead?: OrgPerson;
  members?: OrgPerson[];
  subgroups?: { title: string; members: OrgPerson[] }[];
}

export const ORG_CHAIR: OrgPerson = {
  name: '엄은숙',
  cohort: '4기·중',
  role: '정동회계법인 본부장',
};

export const ORG_GROUPS: readonly OrgGroup[] = [
  {
    name: '이사회',
    lead: { name: '장대용', cohort: '7기·중', role: '부회장' },
    members: [
      { name: '조한준', cohort: '10기·러', role: '총괄이사' },
      { name: '김민정', cohort: '6기·독', role: '(주)크레파스솔루션 대표' },
      { name: '김 웅', cohort: '2기·독', role: '(주)티에스인베스트먼트 대표' },
      { name: '김형주', cohort: '13기·중', role: '(주)봉우 대표' },
      { name: '박승일', cohort: '1기·서', role: '(주)한양특수고무 대표' },
      { name: '안효준', cohort: '15기·불', role: '세하치과 대표원장' },
      { name: '유웅환', cohort: '4기·독', role: '한국벤처투자 前 대표' },
      { name: '이재현', cohort: '7기·서', role: '애플코리아 법무 총괄' },
      { name: '임상진', cohort: '5기·일', role: '(주)데일리비어 대표' },
      { name: '장대용', cohort: '7기·중', role: '(주)엘에스바이오 대표' },
      { name: '조한준', cohort: '10기·러', role: '비아트리스코리아 이사' },
      { name: '허서홍', cohort: '10기·서', role: '(주)GS리테일 대표이사' },
    ],
  },
  {
    name: '감사',
    members: [
      { name: '김민승', cohort: '11기·중', role: '아이니의원 대표원장' },
    ],
  },
  {
    name: '사무국',
    lead: { name: '백강린', cohort: '10기·독', role: '국장' },
    subgroups: [
      {
        title: '책임위원 — 회원',
        members: [
          { name: '신보아', cohort: '16기·중' },
          { name: '정용호', cohort: '14기·독' },
        ],
      },
      {
        title: '책임위원 — 장학사업',
        members: [
          { name: '조한준', cohort: '10기·러' },
          { name: '허정무', cohort: '21기·중' },
        ],
      },
      {
        title: '책임위원 — 자금',
        members: [
          { name: '정재욱', cohort: '10기·독' },
          { name: '이기주', cohort: '12기·불' },
        ],
      },
      {
        title: '책임위원 — 법률',
        members: [
          { name: '이재환', cohort: '10기·독' },
        ],
      },
      {
        title: '책임위원 — 총무',
        members: [
          { name: '김세호', cohort: '15기·중' },
          { name: '황제철', cohort: '29기·독' },
        ],
      },
    ],
  },
  {
    name: '대외협력',
    lead: { name: '박정현', cohort: '8기·독', role: '국장' },
    subgroups: [
      {
        title: '동문협력',
        members: [
          { name: '전혜연', cohort: '6기·일' },
          { name: '정다정', cohort: '10기·독' },
        ],
      },
    ],
  },
  {
    name: '홍보',
    members: [
      { name: '이준용', cohort: '12기·러' },
      { name: '이호진', cohort: '15기·영' },
    ],
  },
];

export interface BusinessItem {
  title: string;
  bullets: string[];
}

export const BUSINESS_HEADLINE = '모교 후배들의 가능성을 여는 장학사업';
export const BUSINESS_SUBHEAD = '지역에서 모교의 위상을 높이는 프로그램을 함께 운영합니다.';

export const BUSINESS_ITEMS: readonly BusinessItem[] = [
  {
    title: "대일외고 학교발전기금 (특별장학금 '동문회 장학금' 지원) 후원",
    bullets: [
      '배움에 열정이 있고 학업 성적이 우수하나 경제적인 어려움을 겪고 있는 모교 학생에게 장학금 지원',
      "'17 상반기 200만원, 하반기 500만원 / '18 상반기 200만원, 하반기 500만원 지급",
      '2022년 2명 학생에게 일시금 400만원, 매월 70만원 지급',
      '연간 신입생 장학금 300만원 지원',
    ],
  },
  {
    title: '대일외고 동문-학생 동행 프로젝트',
    bullets: [
      '동문과 학생이 1:1로 매칭되어 학생의 학업 및 생활을 지원',
      '매월 61명 학생-동문 매칭, 월 305만원 지원',
    ],
  },
  {
    title: '대일외고 학생 급식비 지원',
    bullets: [
      '학생복지심사위원회에서 심사하여 대상자를 선정한 후 급식비 상시 지원',
      "'18년 6월부터 모교 학생 3명에게 매달 1인당 5만원 급식비 지원",
    ],
  },
  {
    title: '대일외고 학생 의복비 지원',
    bullets: [
      '겨울철 추위에 대비해 학생 방한용품 및 의복 구입 비용 지원',
      "'18년 12월 모교 학생 3명에게 1인당 20만원 지원",
    ],
  },
  {
    title: '대일외고 인근 중학교 학생 대상 장학금 지원',
    bullets: [
      '대일외고에 우수한 인재가 지원할 수 있도록 지역 중학교 학생을 대상으로 장학금 지원',
      '성북구·도봉구·강북구·노원구 20개교에 총 400만원 지원',
    ],
  },
  {
    title: '대일외고 학생 실물 지원',
    bullets: [
      '학교와 협의하여 실물 지원이 필요할 때 즉각적으로 지원',
      '동문의 실물 후원 연계 추진',
    ],
  },
];
