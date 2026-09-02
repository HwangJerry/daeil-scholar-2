// AdminLayout — responsive shell with desktop and mobile admin navigation
import { Outlet } from 'react-router-dom';
import { AdminHeader } from './AdminHeader.tsx';
import { AdminSidebar } from './AdminSidebar.tsx';
import { MobileNavDrawer } from './MobileNavDrawer.tsx';

export function AdminLayout() {
  return (
    <div className="flex min-h-screen bg-background">
      <AdminSidebar />
      <MobileNavDrawer />
      <div className="flex flex-1 flex-col">
        <AdminHeader />
        <main className="flex-1 p-6 pb-[calc(5rem+env(safe-area-inset-bottom))] md:pb-6">
          <Outlet />
        </main>
      </div>
    </div>
  );
}
