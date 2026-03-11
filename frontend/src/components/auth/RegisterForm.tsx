// RegisterForm — ID/password registration form with required and optional profile fields.
import { useState } from 'react';
import { usePublicJobCategories } from '../../hooks/usePublicJobCategories';
import { useRegisterFormValidation } from '../../hooks/useRegisterFormValidation';
import { useRegisterSubmit } from '../../hooks/useRegisterSubmit';
import { Button } from '../ui/Button';

const inputClass =
  'w-full rounded-lg border border-border px-3 py-2.5 text-sm outline-none focus:ring-2 focus:ring-primary/20';

export function RegisterForm() {
  const [usrId, setUsrId] = useState('');
  const [password, setPassword] = useState('');
  const [passwordConfirm, setPasswordConfirm] = useState('');
  const [name, setName] = useState('');
  const [phone, setPhone] = useState('');
  const [fn, setFn] = useState('');
  const [email, setEmail] = useState('');
  const [nick, setNick] = useState('');
  const [dept, setDept] = useState('');
  const [jobCat, setJobCat] = useState<number | null>(null);
  const [bizName, setBizName] = useState('');
  const [bizDesc, setBizDesc] = useState('');
  const [bizAddr, setBizAddr] = useState('');
  const [validationError, setValidationError] = useState('');

  const { data: jobCategories = [] } = usePublicJobCategories();
  const { validate } = useRegisterFormValidation();
  const { submit, error: submitError, submitting } = useRegisterSubmit();

  const displayError = validationError || submitError;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const validErr = validate({ usrId, password, passwordConfirm, email });
    if (validErr) {
      setValidationError(validErr);
      return;
    }
    setValidationError('');
    await submit({ usrId, password, name, phone, fn, email, nick, dept, jobCat, bizName, bizDesc, bizAddr });
  };

  return (
    <form onSubmit={handleSubmit} className="space-y-3">
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">아이디 *</label>
        <input
          type="text"
          value={usrId}
          onChange={(e) => setUsrId(e.target.value)}
          required
          className={inputClass}
          placeholder="영문자+숫자, 4~20자"
          autoComplete="username"
        />
      </div>
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">비밀번호 *</label>
        <input
          type="password"
          value={password}
          onChange={(e) => setPassword(e.target.value)}
          required
          className={inputClass}
          placeholder="6자 이상"
          autoComplete="new-password"
        />
      </div>
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">비밀번호 확인 *</label>
        <input
          type="password"
          value={passwordConfirm}
          onChange={(e) => setPasswordConfirm(e.target.value)}
          required
          className={inputClass}
          placeholder="비밀번호를 다시 입력하세요"
          autoComplete="new-password"
        />
      </div>
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">이름 *</label>
        <input
          type="text"
          value={name}
          onChange={(e) => setName(e.target.value)}
          required
          className={inputClass}
          placeholder="실명을 입력하세요"
          autoComplete="name"
        />
      </div>
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">전화번호 *</label>
        <input
          type="tel"
          value={phone}
          onChange={(e) => setPhone(e.target.value)}
          required
          className={inputClass}
          placeholder="010-0000-0000"
          autoComplete="tel"
        />
      </div>
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">기수</label>
        <input
          type="text"
          value={fn}
          onChange={(e) => setFn(e.target.value)}
          className={inputClass}
          placeholder="졸업 기수 (예: 1)"
        />
      </div>
      <div>
        <label className="mb-1 block text-sm font-medium text-text-muted">이메일 *</label>
        <input
          type="email"
          value={email}
          onChange={(e) => setEmail(e.target.value)}
          required
          className={inputClass}
          placeholder="example@email.com"
          autoComplete="email"
        />
      </div>

      <div className="border-t border-border pt-3">
        <p className="mb-3 text-xs text-text-placeholder">
          아래 정보는 선택사항입니다. 가입 후 프로필 편집에서도 입력하실 수 있습니다.
        </p>
        <div className="space-y-3">
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              닉네임 <span className="text-text-placeholder">(선택)</span>
            </label>
            <input
              type="text"
              value={nick}
              onChange={(e) => setNick(e.target.value)}
              className={inputClass}
              placeholder="표시될 닉네임"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              학과/부서 <span className="text-text-placeholder">(선택)</span>
            </label>
            <input
              type="text"
              value={dept}
              onChange={(e) => setDept(e.target.value)}
              className={inputClass}
              placeholder="예: 영어과"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              직업 카테고리 <span className="text-text-placeholder">(선택)</span>
            </label>
            <select
              value={jobCat ?? ''}
              onChange={(e) => setJobCat(e.target.value ? Number(e.target.value) : null)}
              className={inputClass}
            >
              <option value="">선택 안 함</option>
              {jobCategories.map((cat) => (
                <option key={cat.seq} value={cat.seq}>
                  {cat.name}
                </option>
              ))}
            </select>
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              회사명 <span className="text-text-placeholder">(선택)</span>
            </label>
            <input
              type="text"
              value={bizName}
              onChange={(e) => setBizName(e.target.value)}
              className={inputClass}
              placeholder="재직 중인 회사명"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              업무 설명 <span className="text-text-placeholder">(선택)</span>
            </label>
            <input
              type="text"
              value={bizDesc}
              onChange={(e) => setBizDesc(e.target.value)}
              className={inputClass}
              placeholder="예: 소프트웨어 개발"
            />
          </div>
          <div>
            <label className="mb-1 block text-sm font-medium text-text-muted">
              직장 주소 <span className="text-text-placeholder">(선택)</span>
            </label>
            <input
              type="text"
              value={bizAddr}
              onChange={(e) => setBizAddr(e.target.value)}
              className={inputClass}
              placeholder="예: 서울 강남구"
            />
          </div>
        </div>
      </div>

      {displayError && <p className="text-sm text-error-text">{displayError}</p>}

      <Button type="submit" disabled={submitting} className="w-full">
        {submitting ? '가입 신청 중...' : '가입 신청'}
      </Button>
    </form>
  );
}
