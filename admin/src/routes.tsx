// routes — admin SPA route definitions with auth guard and layout wrapper
import { Routes, Route, Navigate } from 'react-router-dom';
import { AdminAuthGuard } from './components/auth/AdminAuthGuard.tsx';
import { AdminLayout } from './components/layout/AdminLayout.tsx';
import { DashboardPage } from './pages/DashboardPage.tsx';
import { NoticeListPage } from './pages/NoticeListPage.tsx';
import { NoticeEditPage } from './pages/NoticeEditPage.tsx';
import { DisclosureListPage } from './pages/DisclosureListPage.tsx';
import { DisclosureEditPage } from './pages/DisclosureEditPage.tsx';
import { BannerAdManagePage } from './pages/BannerAdManagePage.tsx';
import { BannerAdEditPage } from './pages/BannerAdEditPage.tsx';
import { MemberListPage } from './pages/MemberListPage.tsx';
import { MemberDetailPage } from './pages/MemberDetailPage.tsx';
import { PendingMembersPage } from './pages/PendingMembersPage.tsx';
import { AdminLoginPage } from './pages/AdminLoginPage.tsx';
import { JobCategoryPage } from './pages/JobCategoryPage.tsx';
import { DonationMonitorPage } from './pages/DonationMonitorPage.tsx';
import { HistoryManagePage } from './pages/HistoryManagePage.tsx';
import { AppMonitoringPage } from './pages/AppMonitoringPage.tsx';
import { AppSettingsPage } from './pages/AppSettingsPage.tsx';

export function AppRoutes() {
  return (
    <Routes>
      <Route path="/login" element={<AdminLoginPage />} />
      <Route
        element={
          <AdminAuthGuard>
            <AdminLayout />
          </AdminAuthGuard>
        }
      >
        <Route index element={<DashboardPage />} />
        <Route path="notice" element={<NoticeListPage />} />
        <Route path="notice/new" element={<NoticeEditPage />} />
        <Route path="notice/:seq/edit" element={<NoticeEditPage />} />
        <Route path="disclosure" element={<DisclosureListPage />} />
        <Route path="disclosure/new" element={<DisclosureEditPage />} />
        <Route path="disclosure/:seq/edit" element={<DisclosureEditPage />} />
        <Route path="banner-ad" element={<BannerAdManagePage />} />
        <Route path="banner-ad/new" element={<BannerAdEditPage />} />
        <Route path="banner-ad/:bnSeq/edit" element={<BannerAdEditPage />} />
        <Route path="member" element={<MemberListPage />} />
        <Route path="member/pending" element={<PendingMembersPage />} />
        <Route path="member/:seq" element={<MemberDetailPage />} />
        <Route path="job-categories" element={<JobCategoryPage />} />
        <Route path="history" element={<HistoryManagePage />} />
        <Route path="donation" element={<DonationMonitorPage />} />
        <Route path="app-monitoring" element={<AppMonitoringPage />} />
        <Route path="app-settings" element={<AppSettingsPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
