// AdminLayout — shell with sidebar navigation and header wrapping page content
import { Outlet } from 'react-router-dom';
import { AdminHeader } from './AdminHeader.tsx';
import { AdminSidebar } from './AdminSidebar.tsx';

export function AdminLayout() {
  return (
    <div className="flex min-h-screen bg-background">
      <AdminSidebar />
      <div className="flex flex-1 flex-col">
        <AdminHeader />
        <main className="flex-1 p-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
