import { BusinessEventsSection } from '../components/monitoring/BusinessEventsSection.tsx';
import { SentryMonitoringSection } from '../components/monitoring/SentryMonitoringSection.tsx';

export function AppMonitoringPage() {
  return (
    <div className="space-y-8">
      <h2 className="text-xl font-bold text-dark-slate">앱 모니터링</h2>
      <SentryMonitoringSection />
      <BusinessEventsSection />
    </div>
  );
}
