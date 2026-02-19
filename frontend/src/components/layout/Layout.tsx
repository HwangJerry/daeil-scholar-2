// Layout — App shell with top/bottom navigation and responsive content area
import { Outlet } from "react-router-dom";
import BottomNav from "./BottomNav";
import TopNav from "./TopNav";

export default function Layout() {
  return (
    <div className="min-h-screen bg-background font-sans">
      <TopNav />
      <main className="container-app md:container md:mx-auto md:max-w-[1080px] md:shadow-none md:bg-transparent md:pt-6 pb-28 md:pb-12 px-0 md:px-6">
        <Outlet />
      </main>
      <BottomNav />
    </div>
  );
}
