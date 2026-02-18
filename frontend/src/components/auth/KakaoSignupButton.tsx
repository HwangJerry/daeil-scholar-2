// KakaoSignupButton — Outline-style Kakao OAuth button for new user registration.
export function KakaoSignupButton() {
  const handleClick = () => {
    window.location.href = '/api/auth/kakao';
  };

  return (
    <button
      onClick={handleClick}
      className="flex h-12 w-full items-center justify-center gap-2 rounded-lg border-2 border-kakao bg-white font-medium text-kakao-text transition-all duration-150 hover:bg-kakao/10 active:scale-[0.98]"
    >
      <svg className="h-5 w-5" viewBox="0 0 24 24" fill="currentColor">
        <path d="M12 3C6.48 3 2 6.48 2 10.5c0 2.57 1.68 4.83 4.22 6.13l-1.07 3.96c-.09.33.27.6.56.42L9.7 18.3c.74.12 1.51.2 2.3.2 5.52 0 10-3.48 10-7.5S17.52 3 12 3z" />
      </svg>
      카카오로 회원가입
    </button>
  );
}
