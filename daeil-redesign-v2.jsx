import { useState, useEffect, useRef, useCallback } from "react";

// ── Mock data ──────────────────────────────────────────────
const MAIN_POST = {
  id: 0,
  title: "2026년 신년사",
  excerpt:
    "새해를 맞아 대일외고 장학회 동문 여러분께 전하는 회장의 신년 인사말입니다. 올 한 해도 함께 성장하고 나누는 공동체가 되길 바랍니다.",
  date: "2026-01-01T10:41:53",
  views: 22,
  category: "공지",
  author: "회장 박민준 (92학번)",
};

const FEED_DATA = [
  { id: 1, title: "제5차 그랜드마스터클래스 안내", excerpt: "각 분야 선배 동문들이 직접 강의하는 커리어 멘토링 프로그램. 신청 마감 D-7.", date: "2025-12-16T00:32:57", views: 93, category: "이벤트" },
  { id: 2, title: "2025년 12월 장학회 소식", excerpt: "이번 달 장학금 수혜자 발표 및 신규 장학생 모집 일정을 공유합니다.", date: "2025-12-10T14:20:00", views: 157, category: "장학" },
  { id: 3, title: "동문 기업 채용 연계 프로그램", excerpt: "재학생과 졸업생을 위한 동문 기업 인턴십 및 취업 연계 기회를 안내합니다.", date: "2025-11-28T09:00:00", views: 214, category: "커리어" },
  { id: 4, title: "2025 하반기 장학생 면접 일정 공고", excerpt: "장학생 선발 면접 일정 및 장소가 확정되었습니다. 대상자는 반드시 확인 바랍니다.", date: "2025-11-15T11:00:00", views: 88, category: "장학" },
  { id: 5, title: "동문 창업 네트워킹 데이", excerpt: "창업한 동문들을 위한 네트워킹 행사를 개최합니다. 선배-후배 간 노하우를 나누는 자리입니다.", date: "2025-11-01T18:00:00", views: 302, category: "이벤트" },
  { id: 6, title: "11월 정기 이사회 결과 안내", excerpt: "2025년 11월 정기 이사회 의결 사항 및 주요 결정 내용을 공유합니다.", date: "2025-10-30T10:00:00", views: 45, category: "공지" },
  { id: 7, title: "대일외고 개교 기념일 특집 인터뷰", excerpt: "다양한 분야에서 활동 중인 동문들의 학창 시절 이야기와 현재를 담았습니다.", date: "2025-10-22T09:30:00", views: 412, category: "커리어" },
  { id: 8, title: "2025 장학기금 중간 결산 보고", excerpt: "올해 1~3분기 기부 현황과 장학금 지급 내역을 투명하게 공개합니다.", date: "2025-10-05T14:00:00", views: 178, category: "장학" },
];

const PAGE_SIZE = 4;

const categoryStyle = {
  공지: { bg: "#EFF2FF", text: "#3B5BDB", border: "#C5D0FA" },
  이벤트: { bg: "#FFF9DB", text: "#E67700", border: "#FFE066" },
  장학: { bg: "#EBFBEE", text: "#2B8A3E", border: "#8CE99A" },
  커리어: { bg: "#F3F0FF", text: "#6741D9", border: "#D0BFFF" },
};

// ── Helpers ────────────────────────────────────────────────
function formatDate(iso) {
  const d = new Date(iso);
  return `${d.getFullYear()}. ${String(d.getMonth() + 1).padStart(2, "0")}. ${String(d.getDate()).padStart(2, "0")}`;
}

// ── Sub-components ─────────────────────────────────────────

function CategoryBadge({ cat }) {
  const s = categoryStyle[cat] || categoryStyle["공지"];
  return (
    <span
      style={{
        background: s.bg,
        color: s.text,
        border: `1px solid ${s.border}`,
        fontFamily: "'DM Sans', sans-serif",
        fontSize: "10px",
        letterSpacing: "0.08em",
        textTransform: "uppercase",
        padding: "2px 8px",
        borderRadius: "999px",
        fontWeight: 500,
      }}
    >
      {cat}
    </span>
  );
}

function EyeIcon() {
  return (
    <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" style={{ display: "inline" }}>
      <path d="M1 12s4-8 11-8 11 8 11 8-4 8-11 8-11-8-11-8z" />
      <circle cx="12" cy="12" r="3" />
    </svg>
  );
}

// ── Main Post (Pinned) ─────────────────────────────────────
function MainPostCard() {
  return (
    <div
      style={{
        background: "linear-gradient(140deg, #0f1b35 0%, #1a2f5a 55%, #1e3a6e 100%)",
        borderRadius: "20px",
        padding: "36px",
        position: "relative",
        overflow: "hidden",
        cursor: "pointer",
        minHeight: "260px",
        display: "flex",
        flexDirection: "column",
        justifyContent: "space-between",
      }}
    >
      {/* Ambient glow */}
      <div style={{
        position: "absolute", top: "-40px", right: "-40px",
        width: "200px", height: "200px",
        background: "radial-gradient(circle, rgba(100,140,255,0.12) 0%, transparent 70%)",
        pointerEvents: "none",
      }} />
      <div style={{
        position: "absolute", bottom: "-20px", left: "30px",
        width: "160px", height: "160px",
        background: "radial-gradient(circle, rgba(80,180,255,0.07) 0%, transparent 70%)",
        pointerEvents: "none",
      }} />

      {/* Top area */}
      <div style={{ display: "flex", alignItems: "flex-start", justifyContent: "space-between" }}>
        <div style={{ display: "flex", alignItems: "center", gap: "8px" }}>
          {/* Pin icon */}
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="rgba(255,255,255,0.45)" strokeWidth="2">
            <path d="M12 2L12 10M12 10L8 6M12 10L16 6M5 12l1 9h12l1-9H5zM3 12h18"/>
          </svg>
          <span style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "10px", letterSpacing: "0.12em", textTransform: "uppercase", color: "rgba(255,255,255,0.4)" }}>
            고정 공지
          </span>
        </div>
        <CategoryBadge cat={MAIN_POST.category} />
      </div>

      {/* Content */}
      <div>
        <h2 style={{
          fontFamily: "'Noto Serif KR', serif",
          fontSize: "24px", fontWeight: 700,
          color: "#FFFFFF", lineHeight: 1.35,
          marginBottom: "12px",
        }}>
          {MAIN_POST.title}
        </h2>
        <p style={{
          fontFamily: "'DM Sans', sans-serif",
          fontSize: "13px", color: "rgba(255,255,255,0.55)",
          lineHeight: 1.7, marginBottom: "20px",
        }}>
          {MAIN_POST.excerpt}
        </p>
        <div style={{ display: "flex", alignItems: "center", gap: "16px" }}>
          <span style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "12px", color: "rgba(255,255,255,0.35)" }}>
            {formatDate(MAIN_POST.date)}
          </span>
          <span style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "12px", color: "rgba(255,255,255,0.35)", display: "flex", alignItems: "center", gap: "4px" }}>
            <EyeIcon /> {MAIN_POST.views}
          </span>
          <span style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "12px", color: "rgba(255,255,255,0.35)" }}>
            {MAIN_POST.author}
          </span>
        </div>
      </div>
    </div>
  );
}

// ── Donation Widget ────────────────────────────────────────
function DonationWidget() {
  return (
    <div style={{
      background: "#FAFAF8",
      border: "1px solid #E8E5DF",
      borderRadius: "20px",
      padding: "28px",
      display: "flex",
      flexDirection: "column",
      justifyContent: "space-between",
    }}>
      <div>
        <p style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "10px", letterSpacing: "0.12em", textTransform: "uppercase", color: "#AAAAAA", marginBottom: "16px" }}>
          기부 캠페인
        </p>
        <h4 style={{ fontFamily: "'Noto Serif KR', serif", fontSize: "16px", fontWeight: 600, color: "#1a1a2e", marginBottom: "6px" }}>
          2026 장학기금 모금
        </h4>
        <p style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "12px", color: "#999", marginBottom: "20px", lineHeight: 1.6 }}>
          목표 2억원 · 65명 참여
        </p>

        {/* Amount */}
        <div style={{ marginBottom: "10px" }}>
          <div style={{ display: "flex", justifyContent: "space-between", marginBottom: "8px" }}>
            <span style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "22px", fontWeight: 700, color: "#1a1a2e", letterSpacing: "-0.5px" }}>
              8,957<span style={{ fontSize: "13px", fontWeight: 400, color: "#999", marginLeft: "2px" }}>만원</span>
            </span>
            <span style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "13px", color: "#AAAAAA", alignSelf: "flex-end", paddingBottom: "3px" }}>
              44.8%
            </span>
          </div>
          {/* Progress bar */}
          <div style={{ height: "6px", background: "#EEEAE4", borderRadius: "99px", overflow: "hidden" }}>
            <div style={{
              width: "44.8%", height: "100%",
              background: "linear-gradient(90deg, #1a1a2e, #2d3a6e)",
              borderRadius: "99px",
              animation: "fillBar 1.4s cubic-bezier(0.4,0,0.2,1) forwards",
              animationDelay: "0.5s",
            }} />
          </div>
          <p style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "11px", color: "#C0BBB5", marginTop: "6px", textAlign: "right" }}>
            목표 2억원
          </p>
        </div>
      </div>

      <button
        style={{
          width: "100%", padding: "12px",
          borderRadius: "12px",
          border: "1.5px solid #1a1a2e",
          background: "transparent",
          color: "#1a1a2e",
          fontFamily: "'DM Sans', sans-serif",
          fontSize: "13px", fontWeight: 600,
          cursor: "pointer",
          transition: "all 0.2s ease",
          marginTop: "20px",
        }}
        onMouseEnter={e => { e.target.style.background = "#1a1a2e"; e.target.style.color = "#fff"; }}
        onMouseLeave={e => { e.target.style.background = "transparent"; e.target.style.color = "#1a1a2e"; }}
      >
        기부하기
      </button>
    </div>
  );
}

// ── Network Widget ─────────────────────────────────────────
const AVATARS = [
  { label: "이", color: "#6B7CFF" },
  { label: "김", color: "#FF8C6B" },
  { label: "박", color: "#4DB6AC" },
  { label: "최", color: "#FFB74D" },
  { label: "정", color: "#BA68C8" },
];

function NetworkWidget() {
  return (
    <div style={{
      background: "#FAFAF8",
      border: "1px solid #E8E5DF",
      borderRadius: "20px",
      padding: "24px",
    }}>
      <p style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "10px", letterSpacing: "0.12em", textTransform: "uppercase", color: "#AAAAAA", marginBottom: "16px" }}>
        동문 네트워크
      </p>
      {/* Avatar stack */}
      <div style={{ display: "flex", alignItems: "center", marginBottom: "14px" }}>
        {AVATARS.map((a, i) => (
          <div key={i} style={{
            width: "32px", height: "32px",
            borderRadius: "50%",
            background: a.color,
            border: "2px solid #FAFAF8",
            marginLeft: i > 0 ? "-8px" : "0",
            zIndex: 5 - i,
            position: "relative",
            display: "flex", alignItems: "center", justifyContent: "center",
            color: "#fff",
            fontSize: "11px", fontWeight: 700,
            fontFamily: "'Noto Serif KR', serif",
          }}>
            {a.label}
          </div>
        ))}
        <span style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "12px", color: "#888", marginLeft: "10px" }}>
          +2,842명
        </span>
      </div>
      {/* Stats row */}
      <div style={{ display: "flex", gap: "0", marginBottom: "18px", background: "#F0EDE8", borderRadius: "12px", overflow: "hidden" }}>
        {[["전체 동문", "2,847"], ["이번 주 가입", "12"]].map(([label, val], i) => (
          <div key={i} style={{
            flex: 1, padding: "12px",
            borderRight: i === 0 ? "1px solid #E8E5DF" : "none",
            textAlign: "center",
          }}>
            <p style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "16px", fontWeight: 700, color: "#1a1a2e", marginBottom: "2px" }}>{val}</p>
            <p style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "10px", color: "#AAAAAA" }}>{label}</p>
          </div>
        ))}
      </div>
      <button
        style={{
          width: "100%", padding: "11px",
          borderRadius: "12px",
          border: "none",
          background: "#F0EDE8",
          color: "#1a1a2e",
          fontFamily: "'DM Sans', sans-serif",
          fontSize: "13px", fontWeight: 500,
          cursor: "pointer",
          transition: "background 0.2s",
        }}
        onMouseEnter={e => e.target.style.background = "#E5E1D8"}
        onMouseLeave={e => e.target.style.background = "#F0EDE8"}
      >
        동문 찾기 →
      </button>
    </div>
  );
}

// ── Feed Post Card ─────────────────────────────────────────
function FeedCard({ post, style }) {
  const [hovered, setHovered] = useState(false);
  return (
    <div
      onMouseEnter={() => setHovered(true)}
      onMouseLeave={() => setHovered(false)}
      style={{
        background: "#FAFAF8",
        border: `1px solid ${hovered ? "#C8C5BF" : "#E8E5DF"}`,
        borderRadius: "16px",
        padding: "24px",
        cursor: "pointer",
        transition: "border-color 0.2s, box-shadow 0.2s",
        boxShadow: hovered ? "0 2px 12px rgba(0,0,0,0.06)" : "none",
        position: "relative",
        ...style,
      }}
    >
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-start", gap: "12px" }}>
        <div style={{ flex: 1, minWidth: 0 }}>
          <div style={{ display: "flex", alignItems: "center", gap: "8px", marginBottom: "10px" }}>
            <CategoryBadge cat={post.category} />
            <span style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "12px", color: "#AAAAAA" }}>
              {formatDate(post.date)}
            </span>
          </div>
          <h3 style={{
            fontFamily: "'Noto Serif KR', serif",
            fontSize: "15px", fontWeight: 600,
            color: "#1a1a2e", marginBottom: "6px",
            lineHeight: 1.45,
          }}>
            {post.title}
          </h3>
          <p style={{
            fontFamily: "'DM Sans', sans-serif",
            fontSize: "13px", color: "#888",
            lineHeight: 1.65,
            overflow: "hidden",
            display: "-webkit-box",
            WebkitLineClamp: 2,
            WebkitBoxOrient: "vertical",
          }}>
            {post.excerpt}
          </p>
        </div>
        <div style={{
          color: "#AAAAAA",
          fontFamily: "'DM Sans', sans-serif",
          fontSize: "18px",
          flexShrink: 0,
          marginTop: "2px",
          transform: hovered ? "translateX(3px)" : "translateX(-1px)",
          opacity: hovered ? 1 : 0,
          transition: "transform 0.2s, opacity 0.2s",
        }}>
          →
        </div>
      </div>
      {/* Footer */}
      <div style={{
        display: "flex", alignItems: "center", gap: "12px",
        marginTop: "14px", paddingTop: "14px",
        borderTop: "1px solid #F0EDE8",
        fontFamily: "'DM Sans', sans-serif", fontSize: "12px", color: "#AAAAAA",
      }}>
        <span style={{ display: "flex", alignItems: "center", gap: "4px" }}>
          <EyeIcon /> {post.views}
        </span>
      </div>
    </div>
  );
}

// ── Infinite Scroll Feed ───────────────────────────────────
function InfiniteFeed() {
  const [items, setItems] = useState(FEED_DATA.slice(0, PAGE_SIZE));
  const [page, setPage] = useState(1);
  const [loading, setLoading] = useState(false);
  const [hasMore, setHasMore] = useState(FEED_DATA.length > PAGE_SIZE);
  const loaderRef = useRef(null);

  const loadMore = useCallback(() => {
    if (loading || !hasMore) return;
    setLoading(true);
    setTimeout(() => {
      const nextPage = page + 1;
      const next = FEED_DATA.slice(0, nextPage * PAGE_SIZE);
      setItems(next);
      setPage(nextPage);
      setHasMore(next.length < FEED_DATA.length);
      setLoading(false);
    }, 600);
  }, [loading, hasMore, page]);

  useEffect(() => {
    const observer = new IntersectionObserver(
      (entries) => { if (entries[0].isIntersecting) loadMore(); },
      { threshold: 0.1 }
    );
    if (loaderRef.current) observer.observe(loaderRef.current);
    return () => observer.disconnect();
  }, [loadMore]);

  return (
    <div>
      <div style={{ display: "flex", flexDirection: "column", gap: "12px" }}>
        {items.map((post, i) => (
          <FeedCard key={post.id} post={post} style={{ animationDelay: `${i * 0.05}s` }} />
        ))}
      </div>

      {/* Loader sentinel */}
      <div ref={loaderRef} style={{ height: "40px", display: "flex", alignItems: "center", justifyContent: "center", marginTop: "16px" }}>
        {loading && (
          <div style={{ display: "flex", gap: "6px" }}>
            {[0, 1, 2].map(i => (
              <div key={i} style={{
                width: "6px", height: "6px", borderRadius: "50%",
                background: "#C8C5BF",
                animation: `bounce 0.8s ease infinite`,
                animationDelay: `${i * 0.15}s`,
              }} />
            ))}
          </div>
        )}
        {!hasMore && !loading && (
          <p style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "12px", color: "#C0BBB5" }}>
            모든 게시글을 불러왔습니다
          </p>
        )}
      </div>
    </div>
  );
}

// ── App ────────────────────────────────────────────────────
export default function DaeilRedesignV2() {
  const [activeNav, setActiveNav] = useState("뉴스피드");

  return (
    <div style={{ fontFamily: "'Noto Serif KR', serif", background: "#F5F4F0", minHeight: "100vh" }}>
      <style>{`
        @import url('https://fonts.googleapis.com/css2?family=Noto+Serif+KR:wght@400;500;600;700&family=DM+Sans:ital,wght@0,300;0,400;0,500;0,600;1,400&display=swap');
        * { box-sizing: border-box; margin: 0; padding: 0; }
        @keyframes fillBar { from { width: 0% } to { width: 44.8% } }
        @keyframes bounce {
          0%, 80%, 100% { transform: scale(0.8); opacity: 0.4; }
          40% { transform: scale(1.2); opacity: 1; }
        }
        @keyframes fadeUp {
          from { opacity: 0; transform: translateY(12px); }
          to { opacity: 1; transform: translateY(0); }
        }
        .section-fadein { animation: fadeUp 0.5s ease forwards; }
      `}</style>

      {/* ── HEADER ── */}
      <header style={{
        background: "#FAFAF8",
        borderBottom: "1px solid #E8E5DF",
        position: "sticky", top: 0, zIndex: 100,
      }}>
        <div style={{ maxWidth: "1080px", margin: "0 auto", padding: "0 24px", height: "56px", display: "flex", alignItems: "center", justifyContent: "space-between" }}>
          <div style={{ display: "flex", alignItems: "center", gap: "10px" }}>
            <div style={{ width: "28px", height: "28px", borderRadius: "50%", background: "#1a1a2e", display: "flex", alignItems: "center", justifyContent: "center" }}>
              <span style={{ color: "#fff", fontSize: "11px", fontWeight: 700, fontFamily: "'DM Sans', sans-serif" }}>D</span>
            </div>
            <span style={{ fontFamily: "'Noto Serif KR', serif", fontSize: "13px", fontWeight: 600, color: "#1a1a2e", letterSpacing: "-0.2px" }}>
              대일외고 장학회
            </span>
          </div>

          <nav style={{ display: "flex", gap: "28px" }}>
            {["뉴스피드", "동문찾기", "기부", "쪽지", "마이페이지"].map((item) => (
              <button key={item} onClick={() => setActiveNav(item)} style={{
                background: "none", border: "none", cursor: "pointer",
                fontFamily: "'DM Sans', sans-serif",
                fontSize: "13px",
                color: activeNav === item ? "#1a1a2e" : "#AAAAAA",
                fontWeight: activeNav === item ? 600 : 400,
                padding: "4px 0",
                borderBottom: activeNav === item ? "1.5px solid #1a1a2e" : "1.5px solid transparent",
                transition: "all 0.2s",
              }}>
                {item}
              </button>
            ))}
          </nav>
        </div>
      </header>

      {/* ── MAIN ── */}
      <main style={{ maxWidth: "1080px", margin: "0 auto", padding: "32px 24px" }}>

        {/* Section label */}
        <div style={{ marginBottom: "20px" }} className="section-fadein">
          <p style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "10px", letterSpacing: "0.14em", textTransform: "uppercase", color: "#AAAAAA", marginBottom: "4px" }}>
            최신 소식
          </p>
          <h1 style={{ fontFamily: "'Noto Serif KR', serif", fontSize: "22px", fontWeight: 700, color: "#1a1a2e" }}>
            뉴스피드
          </h1>
        </div>

        {/* ── SINGLE 2+1 GRID wrapping everything ── */}
        <div style={{ display: "grid", gridTemplateColumns: "1fr 360px", gap: "16px", alignItems: "start" }}>

          {/* LEFT COLUMN: pinned post + feed */}
          <div>
            {/* Main (pinned) post */}
            <div style={{ marginBottom: "16px" }} className="section-fadein">
              <MainPostCard />
            </div>

            {/* Feed header */}
            <div style={{ display: "flex", alignItems: "center", gap: "12px", marginBottom: "14px" }}>
              <p style={{ fontFamily: "'DM Sans', sans-serif", fontSize: "12px", color: "#AAAAAA", whiteSpace: "nowrap" }}>
                최신순
              </p>
              <div style={{ height: "1px", flex: 1, background: "#E8E5DF" }} />
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="#AAAAAA" strokeWidth="2">
                <line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/>
                <line x1="8" y1="18" x2="21" y2="18"/><line x1="3" y1="6" x2="3.01" y2="6"/>
                <line x1="3" y1="12" x2="3.01" y2="12"/><line x1="3" y1="18" x2="3.01" y2="18"/>
              </svg>
            </div>

            {/* Infinite feed */}
            <InfiniteFeed />
          </div>

          {/* RIGHT COLUMN: sticky sidebar with both widgets */}
          <div style={{ position: "sticky", top: "72px", display: "flex", flexDirection: "column", gap: "14px" }}>
            <DonationWidget />
            <NetworkWidget />
          </div>

        </div>
      </main>
    </div>
  );
}
