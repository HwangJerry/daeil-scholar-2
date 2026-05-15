// routes — admin SPA route definitions with auth guard and layout wrapper
import { Routes, Route, Navigate } from 'react-router-dom';
import { AdminAuthGuard } from './components/auth/AdminAuthGuard.tsx';
import { AdminLayout } from './components/layout/AdminLayout.tsx';
import { DashboardPage } from './pages/DashboardPage.tsx';
import { NoticeListPage } from './pages/NoticeListPage.tsx';
import { NoticeEditPage } from './pages/NoticeEditPage.tsx';
import { DisclosureListPage } from './pages/DisclosureListPage.tsx';
import { DisclosureEditPage } from './pages/DisclosureEditPage.tsx';
import { AdManagePage } from './pages/AdManagePage.tsx';
import { AdEditPage } from './pages/AdEditPage.tsx';
import { MemberListPage } from './pages/MemberListPage.tsx';
import { MemberDetailPage } from './pages/MemberDetailPage.tsx';
import { PendingMembersPage } from './pages/PendingMembersPage.tsx';
import { AdminLoginPage } from './pages/AdminLoginPage.tsx';
import { JobCategoryPage } from './pages/JobCategoryPage.tsx';
import { DonationMonitorPage } from './pages/DonationMonitorPage.tsx';
import { HistoryManagePage } from './pages/HistoryManagePage.tsx';

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
        <Route path="ad" element={<AdManagePage />} />
        <Route path="ad/new" element={<AdEditPage />} />
        <Route path="ad/:maSeq/edit" element={<AdEditPage />} />
        <Route path="member" element={<MemberListPage />} />
        <Route path="member/pending" element={<PendingMembersPage />} />
        <Route path="member/:seq" element={<MemberDetailPage />} />
        <Route path="job-categories" element={<JobCategoryPage />} />
        <Route path="history" element={<HistoryManagePage />} />
        <Route path="donation" element={<DonationMonitorPage />} />
      </Route>
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
