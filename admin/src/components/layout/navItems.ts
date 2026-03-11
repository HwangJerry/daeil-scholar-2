// navItems — shared navigation item definitions for sidebar and mobile drawer
import { LayoutDashboard, FileText, Megaphone, Heart, Users, UserCheck } from 'lucide-react';

export const NAV_ITEMS = [
  { to: '/', icon: LayoutDashboard, label: '대시보드', end: true },
  { to: '/notice', icon: FileText, label: '공지 관리', end: false },
  { to: '/ad', icon: Megaphone, label: '광고 관리', end: false },
  { to: '/donation', icon: Heart, label: '기부 현황', end: true },
  { to: '/member', icon: Users, label: '회원 관리', end: false },
  { to: '/member/pending', icon: UserCheck, label: '가입 신청', end: true },
] as const;
