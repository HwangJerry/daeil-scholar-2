// OrganizationTree — Desktop hierarchy diagram: chair -> 5 groups -> sub-teams
import { UserRound } from 'lucide-react';
import {
  ORG_CHAIR,
  ORG_GROUPS,
  type OrgGroup,
  type OrgPerson,
} from '../../constants/aboutContent';
import { cn } from '../../lib/utils';
import { Card } from '../ui/Card';

const TREE_GROUP_ORDER = ['대외협력', '이사회', '감사', '홍보', '사무국'] as const;

function findGroup(name: string): OrgGroup {
  const group = ORG_GROUPS.find((candidate) => candidate.name === name);
  if (!group) throw new Error(`Unknown org group: ${name}`);
  return group;
}

function PersonLine({ person }: { person: OrgPerson }) {
  return (
    <p
      className={cn(
        'flex min-w-0 flex-wrap items-baseline gap-x-1.5 gap-y-0.5 text-[11px] leading-5 text-text-secondary [overflow-wrap:anywhere]',
      )}
    >
      <span className={cn('font-serif font-semibold text-text-primary')}>{person.name}</span>
      <span className={cn('text-text-tertiary')}>({person.cohort})</span>
      {person.role && <span className={cn('min-w-0')}>{person.role}</span>}
    </p>
  );
}

function FeaturedPeople({ people }: { people: OrgPerson[] }) {
  if (people.length === 0) return null;

  return (
    <div className={cn('flex flex-wrap items-center gap-x-3 gap-y-1.5')}>
      {people.map((person, index) => (
        <div
          key={`${person.name}-${person.role ?? person.cohort}`}
          className={cn('flex min-w-0 items-center gap-3')}
        >
          {index > 0 && (
            <span aria-hidden="true" className={cn('h-3.5 w-px shrink-0 bg-border-hover')} />
          )}
          <p className={cn('text-[13px] font-serif font-semibold text-text-primary [overflow-wrap:anywhere]')}>
            <span>{person.name}</span> · {person.role ?? person.cohort}
          </p>
        </div>
      ))}
    </div>
  );
}

function GroupMemberGrid({ members, groupName }: { members: OrgPerson[]; groupName: string }) {
  if (members.length === 0) return null;

  return (
    <ul className={cn('mt-3 grid grid-cols-2 gap-x-3 gap-y-1.5 border-t border-border-subtle pt-3')}>
      {members.map((member, index) => (
        <li
          key={`${groupName}-${member.name}-${member.role ?? member.cohort}-${index}`}
          className={cn('min-w-0')}
        >
          <PersonLine person={member} />
        </li>
      ))}
    </ul>
  );
}

function GroupSubteams({ group }: { group: OrgGroup }) {
  if (!group.subgroups || group.subgroups.length === 0) return null;

  return (
    <div className={cn('mt-3 flex flex-col items-center gap-3')}>
      <span aria-hidden="true" className={cn('h-4 w-px shrink-0 bg-border')} />
      <div className={cn('flex w-full flex-col gap-2.5')}>
        {group.subgroups.map((subgroup) => (
          <section
            key={subgroup.title}
            aria-labelledby={`tree-${group.name}-${subgroup.title}`}
            className={cn('rounded-lg border border-border-subtle bg-background px-3 py-2.5')}
          >
            <h5
              id={`tree-${group.name}-${subgroup.title}`}
              className={cn('text-[11px] font-semibold text-text-primary [overflow-wrap:anywhere]')}
            >
              {subgroup.title}
            </h5>
            <ul className={cn('mt-1.5 space-y-1')}>
              {subgroup.members.map((member) => (
                <li key={`${subgroup.title}-${member.name}`}>
                  <PersonLine person={member} />
                </li>
              ))}
            </ul>
          </section>
        ))}
      </div>
    </div>
  );
}

function GroupNode({ group }: { group: OrgGroup }) {
  const featuredPeople = group.lead
    ? [group.lead, ...(group.coLead ? [group.coLead] : [])]
    : group.subgroups
      ? []
      : (group.members ?? []);
  const gridMembers =
    group.lead && group.members ? group.members.filter((member) => member !== group.coLead) : [];

  return (
    <div className={cn('flex min-w-0 flex-col items-center')}>
      <span aria-hidden="true" className={cn('h-6 w-px shrink-0 bg-border')} />
      <Card variant="default" padding="md" className={cn('w-full min-w-0')}>
        <h4 className={cn('font-serif text-[15px] font-bold text-text-primary')}>{group.name}</h4>
        {featuredPeople.length > 0 && (
          <div className={cn('mt-2')}>
            <FeaturedPeople people={featuredPeople} />
          </div>
        )}
        <GroupMemberGrid members={gridMembers} groupName={group.name} />
        <GroupSubteams group={group} />
      </Card>
    </div>
  );
}

export function OrganizationTree() {
  const groups = TREE_GROUP_ORDER.map(findGroup);

  return (
    <div className={cn('flex flex-col items-center')}>
      <Card variant="elevated" padding="none" className={cn('w-full max-w-xs overflow-hidden')}>
        <div className={cn('px-6 py-6 text-center')}>
          <div
            className={cn(
              'mx-auto flex size-11 items-center justify-center rounded-full bg-primary-light text-primary',
            )}
          >
            <UserRound aria-hidden="true" className={cn('size-5')} />
          </div>
          <p className={cn('mt-4 text-xs font-semibold uppercase tracking-[0.2em] text-text-tertiary')}>
            회장
          </p>
          <h3 className={cn('mt-2 font-serif text-xl font-bold leading-tight text-text-primary')}>
            {ORG_CHAIR.name}
          </h3>
          <p className={cn('mt-1 text-xs text-text-tertiary')}>{ORG_CHAIR.cohort}</p>
          {ORG_CHAIR.role && (
            <p className={cn('mx-auto mt-2 max-w-xs text-xs leading-5 text-text-secondary')}>
              {ORG_CHAIR.role}
            </p>
          )}
        </div>
      </Card>

      <span aria-hidden="true" className={cn('h-8 w-px shrink-0 bg-border')} />

      <div className={cn('relative w-full')}>
        <span aria-hidden="true" className={cn('absolute inset-x-[10%] top-0 h-px bg-border')} />
        <ol className={cn('grid grid-cols-5 gap-x-4')}>
          {groups.map((group) => (
            <li key={group.name} className={cn('min-w-0 list-none')}>
              <GroupNode group={group} />
            </li>
          ))}
        </ol>
      </div>
    </div>
  );
}
