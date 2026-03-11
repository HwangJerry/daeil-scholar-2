// Application route definitions — page routes with background-location awareness
import { Routes, Route, Navigate, useLocation } from 'react-router-dom';
import type { Location } from 'react-router-dom';
import Layout from './components/layout/Layout';
import { FeedPage } from './pages/FeedPage';
import { AlumniPage } from './pages/AlumniPage';
import { MyPage } from './pages/MyPage';
import { DonationPage } from './pages/DonationPage';
import { LoginPage } from './pages/LoginPage';
import { LegacyLoginPage } from './pages/LegacyLoginPage';
import { AccountLinkPage } from './pages/AccountLinkPage';
import { PostDetailPage } from './pages/PostDetailPage';
import { DonationResultPage } from './pages/DonationResultPage';
import { MyDonationPage } from './pages/MyDonationPage';
import { SubscriptionPage } from './pages/SubscriptionPage';
import { MessagePage } from './pages/MessagePage';
import { RegisterPage } from './pages/RegisterPage';
import { ForgotPasswordPage } from './pages/ForgotPasswordPage';
import { ResetPasswordPage } from './pages/ResetPasswordPage';
import { ModalRoutes } from './ModalRoutes';

export default function AppRoutes() {
  const location = useLocation();
  const state = location.state as { backgroundLocation?: Location } | null;
  const backgroundLocation = state?.backgroundLocation;

  return (
    <>
      {/* Render background page when a modal is open; otherwise render current location */}
      <Routes location={backgroundLocation ?? location}>
        <Route element={<Layout />}>
          <Route index element={<FeedPage />} />
          <Route path="post/:seq" element={<PostDetailPage />} />
          <Route path="ad/:maSeq" element={<Navigate to="/" replace />} />
          <Route path="alumni" element={<AlumniPage />} />
          <Route path="donation" element={<DonationPage />} />
          <Route path="donation/result" element={<DonationResultPage />} />
          <Route path="me" element={<MyPage />} />
          <Route path="me/donation" element={<MyDonationPage />} />
          <Route path="me/subscription" element={<SubscriptionPage />} />
          <Route path="messages" element={<MessagePage />} />
          <Route path="login" element={<LoginPage />} />
          <Route path="login/legacy" element={<LegacyLoginPage />} />
          <Route path="login/link" element={<AccountLinkPage />} />
          <Route path="register" element={<RegisterPage />} />
          <Route path="forgot-password" element={<ForgotPasswordPage />} />
          <Route path="reset-password" element={<ResetPasswordPage />} />
          <Route path="mypage" element={<MyPage />} />
        </Route>
      </Routes>

      <ModalRoutes />
    </>
  );
}
