// navItems — shared navigation item definitions for sidebar and mobile drawer
import { Activity, LayoutDashboard, FileText, ScrollText, Images, Users, UserCheck, Briefcase, Heart, Clock, Settings2 } from 'lucide-react';

export const NAV_ITEMS = [
  { to: '/', icon: LayoutDashboard, label: '대시보드', end: true },
  { to: '/notice', icon: FileText, label: '공지 관리', end: false },
  { to: '/disclosure', icon: ScrollText, label: '의무공시', end: false },
  { to: '/banner-ad', icon: Images, label: '배너광고 관리', end: false },
  { to: '/donation', icon: Heart, label: '기부 관리', end: true },
  { to: '/member', icon: Users, label: '회원 관리', end: true },
  { to: '/member/pending', icon: UserCheck, label: '가입 신청', end: true },
  { to: '/job-categories', icon: Briefcase, label: '직업 카테고리', end: true },
  { to: '/app-settings', icon: Settings2, label: '앱 설정', end: true },
  { to: '/history', icon: Clock, label: '연혁 관리', end: true },
  { to: '/app-monitoring', icon: Activity, label: '앱 모니터링', end: true },
] as const;
