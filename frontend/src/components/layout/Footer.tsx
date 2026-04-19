// Footer — Site-wide footer with full organization contact details and copyright
export default function Footer() {
  const year = new Date().getFullYear();

  return (
    <footer className="mt-12 border-t border-border-subtle px-6 py-8 text-center">
      <ul className="space-y-1 text-2xs text-text-secondary">
        <li>
          <span className="font-medium">대일외국어고등학교 장학회</span>
          <span className="mx-2 text-border-subtle">|</span>
          <span>Tel: 02-543-3558</span>
          <span className="mx-2 text-border-subtle">|</span>
          <span>Fax: 02-541-1479</span>
        </li>
        <li>
          <span>회장: 엄은숙</span>
          <span className="mx-2 text-border-subtle">|</span>
          <span>총괄이사: 조한준</span>
        </li>
        <li>서울특별시 강남구 언주로 730 동익빌딩 8층</li>
        <li>
          <a
            href="mailto:dflhs.scholar@gmail.com"
            className="underline hover:text-text-primary transition-colors"
          >
            dflhs.scholar@gmail.com
          </a>
          <span className="mx-2 text-border-subtle">|</span>
          <span>고유등록번호: 137-82-84744</span>
        </li>
        <li>기부영수증 발급단체명: 대일외국어고등학교 장학회</li>
      </ul>
      <p className="mt-3 text-2xs text-text-placeholder">
        COPYRIGHT ⓒ {year} 대일외국어고등학교 장학회 ALL RIGHT RESERVED.
      </p>
    </footer>
  );
}
