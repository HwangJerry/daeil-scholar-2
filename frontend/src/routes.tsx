// Application route definitions
import { Routes, Route } from 'react-router-dom';
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

export default function AppRoutes() {
  return (
    <Routes>
      <Route element={<Layout />}>
        <Route index element={<FeedPage />} />
        <Route path="post/:seq" element={<PostDetailPage />} />
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

        {/* Helper redirect for legacy mypage route */}
        <Route path="mypage" element={<MyPage />} />
      </Route>
    </Routes>
  );
}
