// accountErasureLabels — Operator-facing names for erasure tables and hold codes.
const TABLE_LABELS: Record<string, string> = {
  WEO_MEMBER: '회원 기본 정보',
  WEO_MEMBER_SOCIAL: '소셜 로그인 연결',
  ALUMNI_SOCIAL_CREDENTIAL: '소셜 인증 자격',
  WEO_BOARDBBS: '작성한 게시글',
  WEO_BOARDCOMAND: '작성한 댓글',
  WEO_BOARDLIKE: '좋아요',
  ALUMNI_MESSAGE: '주고받은 쪽지',
  ALUMNI_MESSAGE_REPORT: '쪽지 신고 기록',
  ALUMNI_MEMBER_BLOCK: '차단 관계',
  ALUMNI_NOTIFICATION: '알림',
  ALUMNI_PUSH_OUTBOX: '푸시 발송 대기',
  ALUMNI_MOBILE_DEVICE_TOKEN: '푸시 기기 등록',
  ALUMNI_PUSH_DEVICE: '푸시 기기 등록',
  ALUMNI_PUSH_PREFERENCE: '알림 설정',
  ALUMNI_MOBILE_REFRESH_TOKEN: '앱 로그인 세션',
  USER_SESSION: '로그인 세션',
  ALUMNI_PASSWORD_RESET: '비밀번호 재설정 요청',
  WEO_MEMBER_LOG: '로그인 기록',
  WEO_ORDER: '기부 내역',
  WEO_PG_DATA: '결제 상세',
  ALUMNI_DONATION_RETENTION: '기부 보존 결정',
  WEO_FILES: '업로드 파일 정보',
  ALUMNI_UPLOAD_OWNER: '업로드 소유 기록',
  ALUMNI_PROFILE_FILE_HISTORY: '이전 프로필 파일',
  WEO_VISIT_DAILY: '방문 기록',
  ALUMNI_MOBILE_APP_EVENT: '앱 이용 기록',
  ALUMNI_ADMIN_ROLE: '관리자 권한',
  ALUMNI_VERIFICATION: '동문 인증 기록',
  ALUMNI_USER_TAG: '회원 태그',
};

export function tableLabel(table: string): string {
  if (TABLE_LABELS[table]) return TABLE_LABELS[table];
  if (table.startsWith('AUTH_')) return '인증 정보';
  return '기타 회원 기록';
}

export const AUTO_BLOCKERS: Record<string, string> = {
  LEGACY_REPLY_REVIEW_REQUIRED: '과거 게시글에 다른 작성자의 답변이 포함되어 있습니다. 답변 분리와 개인정보 확인 후 재개해주세요.',
  EXTERNAL_HANDOFF_REVIEW_REQUIRED: '내부 삭제 전에 아래 저장소별 담당자와 처리 경로를 확인하고 수동 처리 인계를 기록해주세요.',
  FILE_STILL_REFERENCED: '다른 회원이나 게시글에서 사용하는 파일입니다. 소유권과 공유 참조를 확인해야 합니다.',
  FILE_OWNERSHIP_REVIEW_REQUIRED: '파일을 참조한 기록만 있고 소유권을 확인할 수 없어 삭제를 보류했습니다.',
  EXTERNAL_FILE_HANDOFF_REVIEW_REQUIRED: '기존 파일 큐의 외부 주소를 확인하고 외부 저장소 처리 증빙을 확보해야 합니다.',
  RECEIPT_CONTACT_REVIEW_REQUIRED: '진행 중인 영수증 연락 업무와 원본 보관 위치를 확인해야 합니다.',
  RECEIPT_CONTACT_WORK_PENDING: '영수증 업무·결과 전달·업무용 연락처 정리 확인을 기다리고 있습니다.',
  ERASURE_TARGETS_REVIEW_REQUIRED: '저장소별 작업 등록 상태를 확인해야 합니다.',
  ERASURE_CONTEXT_KEY_REQUIRED: '외부 삭제 작업용 별도 암호화 키 설정이 필요합니다.',
  ERASURE_CONTEXT_UNREADABLE: '외부 삭제 작업의 암호화 정보 확인이 필요합니다.',
  ERASURE_CONTEXT_EXPIRED_REVIEW_REQUIRED: '외부 삭제 작업용 정보의 보관 기한이 지났습니다. 수동 확인이 필요합니다.',
  DONATION_RETENTION_REVIEW_REQUIRED: '기부 자료의 보존 근거와 기간을 확인해야 합니다. 자동 분류 설정이 켜져 있으면 처리 중 자동으로 기록될 수 있습니다.',
  DONATION_ACCOUNT_LINK_REVIEW_REQUIRED: '기부 내역이 다른 회원 계정과도 연결되어 있습니다. 연결 관계를 먼저 정리해야 합니다.',
  INVALID_DONATION_RETENTION_DECISION: '기부 자료의 보존 날짜 또는 근거를 수정해야 합니다.',
  DONATION_ARCHIVE_KEY_REQUIRED: '기부 자료 보관소의 암호화 설정이 필요합니다.',
  EXTERNAL_ERASURE_PROCESSOR_REQUIRED: '외부 서비스·백업 삭제 연동이 필요합니다.',
  EXTERNAL_ERASURE_PENDING: '외부 서비스·백업 삭제 완료를 기다리고 있습니다.',
  PROVIDER_REVOCATION_PENDING: 'Apple·카카오 연결 해제 완료를 기다리고 있습니다.',
  UNHANDLED_ACCOUNT_REFERENCE: '자동 처리 범위 밖에 남은 회원 관련 자료를 확인해야 합니다.',
  BILLING_REVOCATION_REVIEW_REQUIRED: '기존 정기결제 연결을 확인해야 합니다.',
  FILE_PATH_REVIEW_REQUIRED: '파일 경로 또는 소유 관계 확인이 필요합니다.',
  FILE_DELETE_RETRY_REQUIRED: '파일 삭제를 다시 시도합니다.',
  PROVIDER_CREDENTIAL_MISSING: 'Apple 연결 해제에 필요한 인증 정보가 저장돼 있지 않습니다. 해제 방법을 확보해야 처리할 수 있습니다.',
  MEMBER_NOT_WITHDRAWN: '회원 상태가 탈퇴로 바뀌어 있지 않습니다. 신청 상태를 확인해주세요.',
};

export function blockerMessage(code: string): string {
  if (code.startsWith('NON_TRANSACTIONAL_TABLE_')) return `${code.slice('NON_TRANSACTIONAL_TABLE_'.length)} 테이블이 트랜잭션을 지원하지 않아 되돌릴 수 없으므로 처리를 보류합니다. 서버 관리자의 저장 방식 전환이 필요합니다.`;
  return AUTO_BLOCKERS[code] ?? '서버 관리자의 확인이 필요합니다.';
}

const PROVIDER_LABELS: Record<string, string> = { AP: 'Apple', KT: '카카오' };
const UNLINK_STATUS_LABELS: Record<string, string> = {
  pending: '처리 시작 후 해제 요청',
  delivered: '해제 완료',
  failed: '해제 실패 · 재시도 필요',
  missing_credential: '저장된 인증 정보가 없어 해제할 수 없음',
};

export function socialUnlinkLabel(provider: string, status: string): string {
  return `${PROVIDER_LABELS[provider] ?? provider}: ${UNLINK_STATUS_LABELS[status] ?? status}`;
}
