// DonationConfigPage — composes the config form and snapshot history table
import { DonationConfigForm } from '../components/donation/DonationConfigForm.tsx';
import { SnapshotHistoryTable } from '../components/donation/SnapshotHistoryTable.tsx';

export function DonationConfigPage() {
  return (
    <div className="space-y-6">
      <h2 className="text-xl font-bold text-dark-slate">기부 설정</h2>
      <DonationConfigForm />
      <SnapshotHistoryTable />
    </div>
  );
}
