// MessagePage — Tab layout for inbox and outbox messages with auth guard
import { useState } from 'react';
import { AuthGuard } from '../components/auth/AuthGuard';
import { MessageInbox } from '../components/message/MessageInbox';
import { MessageOutbox } from '../components/message/MessageOutbox';

type MessageTab = 'inbox' | 'outbox';

interface TabConfig {
  key: MessageTab;
  label: string;
}

const TABS: TabConfig[] = [
  { key: 'inbox', label: '받은쪽지' },
  { key: 'outbox', label: '보낸쪽지' },
];

function MessagePageContent() {
  const [activeTab, setActiveTab] = useState<MessageTab>('inbox');

  return (
    <div className="px-4 py-6 pb-20">
      <h1 className="text-xl font-bold text-text-primary mb-4 font-serif">쪽지함</h1>

      {/* Tab bar */}
      <div className="flex border-b border-border mb-5">
        {TABS.map((tab) => {
          const isActive = activeTab === tab.key;
          return (
            <button
              key={tab.key}
              onClick={() => setActiveTab(tab.key)}
              className={`relative px-4 py-2.5 text-sm font-medium transition-colors duration-150 ${
                isActive
                  ? 'text-primary'
                  : 'text-text-tertiary hover:text-text-secondary'
              }`}
            >
              {tab.label}
              {isActive && (
                <span className="absolute bottom-0 left-0 right-0 h-0.5 bg-primary rounded-full" />
              )}
            </button>
          );
        })}
      </div>

      {/* Tab content */}
      {activeTab === 'inbox' ? <MessageInbox /> : <MessageOutbox />}
    </div>
  );
}

export function MessagePage() {
  return (
    <AuthGuard>
      <MessagePageContent />
    </AuthGuard>
  );
}
