// HallOfFamePage — switches between the Hall of Fame entry list and spreadsheet upload tabs
import { useState } from 'react';
import { List, Upload } from 'lucide-react';
import { HallOfFameImportSection } from '../components/hallOfFame/HallOfFameImportSection.tsx';
import { HallOfFameListSection } from '../components/hallOfFame/HallOfFameListSection.tsx';
import { Button } from '../components/ui/Button.tsx';

type HallOfFameTab = 'list' | 'upload';

const HALL_OF_FAME_TABS = [
  { id: 'list', label: '목록', icon: List },
  { id: 'upload', label: '엑셀 업로드', icon: Upload },
] as const;

export function HallOfFamePage() {
  const [activeTab, setActiveTab] = useState<HallOfFameTab>('list');

  return (
    <div className="space-y-5">
      <div>
        <h2 className="text-xl font-bold text-dark-slate">명예의 전당</h2>
        <p className="mt-1 text-sm text-cool-gray">앱에 공개되는 기부자 명단을 관리합니다.</p>
      </div>

      <nav
        role="tablist"
        aria-label="명예의 전당 메뉴"
        className="flex gap-1 rounded-2xl border border-border-light bg-white p-1.5 shadow-sm"
      >
        {HALL_OF_FAME_TABS.map((tab) => {
          const Icon = tab.icon;
          const isActive = activeTab === tab.id;
          return (
            <Button
              key={tab.id}
              id={`hall-tab-${tab.id}`}
              role="tab"
              type="button"
              variant={isActive ? 'secondary' : 'ghost'}
              aria-selected={isActive}
              aria-controls={`hall-panel-${tab.id}`}
              className="flex-1 gap-2 rounded-xl px-3"
              onClick={() => setActiveTab(tab.id)}
            >
              <Icon className="h-4 w-4" />
              {tab.label}
            </Button>
          );
        })}
      </nav>

      <div
        id="hall-panel-list"
        role="tabpanel"
        aria-labelledby="hall-tab-list"
        hidden={activeTab !== 'list'}
      >
        <HallOfFameListSection />
      </div>
      <div
        id="hall-panel-upload"
        role="tabpanel"
        aria-labelledby="hall-tab-upload"
        hidden={activeTab !== 'upload'}
      >
        <HallOfFameImportSection />
      </div>
    </div>
  );
}
