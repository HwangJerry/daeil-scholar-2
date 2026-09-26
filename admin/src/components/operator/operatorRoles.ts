// operatorRoles — labels and descriptions for root and operator admin roles
import type { AdminRole } from '../../api/operators.ts';

export const ADMIN_ROLE_OPTIONS: { value: AdminRole; label: string; description: string }[] = [
  { value: 'operator', label: '일반 관리자', description: '콘텐츠·회원·기부 관리' },
  { value: 'root', label: 'root 관리자', description: '일반 관리자 권한 + 운영자 관리, 계정 삭제 처리, 탈퇴 회원 기부 증빙' },
];

export function getAdminRoleLabel(role: AdminRole) {
  return ADMIN_ROLE_OPTIONS.find((option) => option.value === role)?.label ?? role;
}
