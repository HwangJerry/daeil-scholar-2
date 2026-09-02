// OrganizationAccordion — Mobile collapsible org chart: chair -> 5 group cards
import { useState } from 'react';
import { ChevronDown, UserRound } from 'lucide-react';
import {
  ORG_CHAIR,
  ORG_GROUPS,
  type OrgGroup,
  type OrgPerson,
} from '../../constants/aboutContent';
import { cn } from '../../lib/utils';
import { Card } from '../ui/Card';

const ACCORDION_GROUP_ORDER = ['대외협력', '이사회', '감사', '홍보', '사무국'] as const;

function findGroup(name: string): OrgGroup {
  const group = ORG_GROUPS.find((candidate) => candidate.name === name);
  if (!group) throw new Error(`Unknown org group: ${name}`);
  return group;
}

function FeaturedPeopleRow({ people }: { people: OrgPerson[] }) {
  if (people.length === 0) return null;

  return (
    <div className={cn('mt-1 flex flex-wrap items-center gap-x-2.5 gap-y-1')}>
      {people.map((person, index) => (
        <div
          key={`${person.name}-${person.role ?? person.cohort}`}
          className={cn('flex min-w-0 items-center gap-2.5')}
        >
          {index > 0 && (
            <span aria-hidden="true" className={cn('h-3 w-px shrink-0 bg-border-hover')} />
          )}
          <span className={cn('text-[12px] font-serif font-semibold text-text-primary [overflow-wrap:anywhere]')}>
            <span>{person.name}</span> · {person.role ?? person.cohort}
          </span>
        </div>
      ))}
    </div>
  );
}

interface GroupCardProps {
  group: OrgGroup;
}

function GroupCard({ group }: GroupCardProps) {
  const [isOpen, setIsOpen] = useState(false);
  const hasSubgroups = Boolean(group.subgroups && group.subgroups.length > 0);
  const featuredPeople = group.lead
    ? [group.lead, ...(group.coLead ? [group.coLead] : [])]
    : group.subgroups
      ? []
      : (group.members ?? []);
  // Only 이사회 has both a lead and a flat members array today — the count
  // label below is specific to that case (see aboutContent.ts ORG_GROUPS).
  // Counts every member.length entry, including the featured coLead — they
  // are still a director, just called out separately alongside the lead.
  const memberCount =
    group.lead && group.members
      ? group.members.length
      : undefined;

  return (
    <Card variant="default" padding="none" className={cn('overflow-hidden')}>
      <button
        type="button"
        onClick={hasSubgroups ? () => setIsOpen((open) => !open) : undefined}
        aria-expanded={hasSubgroups ? isOpen : undefined}
        className={cn(
          'flex w-full items-center gap-3 px-4 py-3.5 text-left',
          !hasSubgroups && 'cursor-default',
        )}
      >
        <div className={cn('min-w-0 flex-1')}>
          <h4 className={cn('font-serif text-sm font-bold text-text-primary')}>{group.name}</h4>
          <FeaturedPeopleRow people={featuredPeople} />
          {memberCount !== undefined && (
            <p className={cn('mt-1 text-[11px] text-text-tertiary')}>이사 {memberCount}인</p>
          )}
          {hasSubgroups && (
            <p className={cn('mt-1 text-[11px] text-text-tertiary')}>
              산하 {group.subgroups?.length}개 팀
            </p>
          )}
        </div>
        {hasSubgroups && (
          <ChevronDown
            aria-hidden="true"
            className={cn(
              'size-4 shrink-0 text-text-tertiary transition-transform',
              isOpen && 'rotate-180',
            )}
          />
        )}
      </button>

      {hasSubgroups && isOpen && (
        <div className={cn('space-y-2.5 border-t border-border-subtle px-4 py-3.5')}>
          {group.subgroups?.map((subgroup) => (
            <div key={subgroup.title} className={cn('border-l-2 border-primary-muted pl-3')}>
              <h5 className={cn('text-[11px] font-semibold text-text-primary [overflow-wrap:anywhere]')}>
                {subgroup.title}
              </h5>
              <div className={cn('mt-1 flex flex-wrap gap-x-2.5 gap-y-0.5')}>
                {subgroup.members.map((member) => (
                  <span key={member.name} className={cn('text-[11px] text-text-secondary')}>
                    <span>{member.name}</span> ({member.cohort})
                  </span>
                ))}
              </div>
            </div>
          ))}
        </div>
      )}
    </Card>
  );
}

export function OrganizationAccordion() {
  const groups = ACCORDION_GROUP_ORDER.map(findGroup);

  return (
    <div>
      <Card
        variant="elevated"
        padding="lg"
        className={cn('mx-auto max-w-sm overflow-hidden text-center')}
      >
        <div
          className={cn(
            'mx-auto flex size-10 items-center justify-center rounded-full bg-primary-light text-primary',
          )}
        >
          <UserRound aria-hidden="true" className={cn('size-5')} />
        </div>
        <p className={cn('mt-3 text-xs font-semibold uppercase tracking-[0.2em] text-text-tertiary')}>
          회장
        </p>
        <h3 className={cn('mt-2 font-serif text-lg font-bold leading-tight text-text-primary')}>
          {ORG_CHAIR.name}
        </h3>
        <p className={cn('mt-1 text-xs text-text-tertiary')}>{ORG_CHAIR.cohort}</p>
        {ORG_CHAIR.role && (
          <p className={cn('mt-1 text-xs leading-5 text-text-secondary')}>{ORG_CHAIR.role}</p>
        )}
      </Card>

      <ol className={cn('mt-4 space-y-3')}>
        {groups.map((group) => (
          <li key={group.name} className={cn('list-none')}>
            <GroupCard group={group} />
          </li>
        ))}
      </ol>
    </div>
  );
}
