// navItems — shared navigation item definitions for sidebar and mobile drawer
import { LayoutDashboard, FileText, Megaphone, Heart, Users, UserCheck, Briefcase } from 'lucide-react';

export const NAV_ITEMS = [
  { to: '/', icon: LayoutDashboard, label: '대시보드', end: true },
  { to: '/notice', icon: FileText, label: '공지 관리', end: false },
  { to: '/ad', icon: Megaphone, label: '광고 관리', end: false },
  { to: '/donation', icon: Heart, label: '기부 현황', end: true },
  { to: '/member', icon: Users, label: '회원 관리', end: true },
  { to: '/member/pending', icon: UserCheck, label: '가입 신청', end: true },
  { to: '/job-categories', icon: Briefcase, label: '직업 카테고리', end: true },
] as const;
