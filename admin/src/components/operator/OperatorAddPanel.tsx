// OperatorAddPanel — search an active member by name or phone and grant them an admin role
import { useEffect, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { api } from '../../api/client.ts';
import type { AdminRole } from '../../api/operators.ts';
import type { AdminMemberListItem, AdminMemberListResponse } from '../../types/api.ts';
import { Button } from '../ui/Button.tsx';
import { Input } from '../ui/Input.tsx';
import { Select } from '../ui/Select.tsx';
import { getMemberStatusLabel, isInactiveMemberStatus } from '../member/memberStatus.ts';
import { ADMIN_ROLE_OPTIONS } from './operatorRoles.ts';

const SEARCH_DEBOUNCE_MS = 300;
const SEARCH_RESULT_SIZE = 10;

interface OperatorAddPanelProps {
  existingSeqs: ReadonlySet<number>;
  disabled: boolean;
  onGrant: (member: AdminMemberListItem, role: AdminRole) => void;
}

export function OperatorAddPanel({ existingSeqs, disabled, onGrant }: OperatorAddPanelProps) {
  const [input, setInput] = useState('');
  const [search, setSearch] = useState('');
  const [role, setRole] = useState<AdminRole>('operator');

  useEffect(() => {
    const timer = setTimeout(() => setSearch(input.trim()), SEARCH_DEBOUNCE_MS);
    return () => clearTimeout(timer);
  }, [input]);

  const results = useQuery({
    queryKey: ['admin', 'operator-candidates', search],
    queryFn: () => api.get<AdminMemberListResponse>(
      `/api/admin/member?${new URLSearchParams({ q: search, size: String(SEARCH_RESULT_SIZE) })}`,
    ),
    enabled: search.length > 0,
  });

  return (
    <section className="space-y-3 rounded-2xl border border-border-light bg-white p-5 shadow-sm">
      <h3 className="text-sm font-semibold text-dark-slate">관리자 추가</h3>
      <div className="flex flex-wrap items-center gap-2">
        <Input
          aria-label="회원 검색"
          placeholder="이름 또는 연락처로 검색"
          value={input}
          onChange={(e) => setInput(e.target.value)}
          className="max-w-xs"
        />
        <Select aria-label="부여할 권한" value={role} onChange={(e) => setRole(e.target.value as AdminRole)} className="w-40">
          {ADMIN_ROLE_OPTIONS.map((option) => <option key={option.value} value={option.value}>{option.label}</option>)}
        </Select>
      </div>

      {search && (
        <ul className="divide-y divide-border-light rounded-xl border border-border-light">
          {results.isLoading && <li className="px-3 py-2 text-sm text-cool-gray">검색 중...</li>}
          {results.data?.items.length === 0 && <li className="px-3 py-2 text-sm text-cool-gray">검색 결과가 없습니다.</li>}
          {results.data?.items.map((member) => {
            const isAdmin = existingSeqs.has(member.usrSeq);
            const isInactive = isInactiveMemberStatus(member.usrStatus);
            return (
              <li key={member.usrSeq} className="flex items-center justify-between gap-3 px-3 py-2 text-sm">
                <span className="text-dark-slate">
                  {member.usrName}
                  <span className="ml-2 text-xs text-cool-gray">
                    {member.usrId} · {member.usrFn ? `${member.usrFn}기` : '기수 없음'} · {getMemberStatusLabel(member.usrStatus)}
                  </span>
                </span>
                {isAdmin ? (
                  <span className="text-xs text-cool-gray">이미 관리자</span>
                ) : isInactive ? (
                  <span className="text-xs text-cool-gray">지정 불가</span>
                ) : (
                  <Button size="sm" variant="outline" disabled={disabled} onClick={() => onGrant(member, role)}>
                    추가
                  </Button>
                )}
              </li>
            );
          })}
        </ul>
      )}
    </section>
  );
}
