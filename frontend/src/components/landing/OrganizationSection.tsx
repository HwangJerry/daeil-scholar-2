// OrganizationSection — Hierarchical foundation organization overview for the landing page
import { Network, UserRound } from 'lucide-react';
import {
  ORG_CHAIR,
  ORG_GROUPS,
  type OrgGroup,
  type OrgPerson,
} from '../../constants/aboutContent';
import { cn } from '../../lib/utils';
import { Card } from '../ui/Card';

interface PersonLineProps {
  person: OrgPerson;
  showRole?: boolean;
}

function PersonLine({ person, showRole = true }: PersonLineProps) {
  return (
    <p
      className={cn(
        'flex min-w-0 flex-wrap items-baseline gap-x-2 gap-y-1 text-sm leading-6 [overflow-wrap:anywhere]',
      )}
    >
      <span className={cn('font-serif font-semibold text-text-primary')}>{person.name}</span>
      <span className={cn('text-text-tertiary')}>({person.cohort})</span>
      {showRole && person.role && (
        <span className={cn('min-w-0 text-text-secondary')}>{person.role}</span>
      )}
    </p>
  );
}

function GroupLead({ group }: { group: OrgGroup }) {
  if (!group.lead) return null;

  return (
    <div className={cn('border-l-2 border-primary-muted pl-4')}>
      <p
        className={cn(
          'text-xs font-semibold uppercase tracking-[0.16em] text-text-tertiary [overflow-wrap:anywhere]',
        )}
      >
        {group.lead.role ?? '대표'}
      </p>
      <div className={cn('mt-2')}>
        <PersonLine person={group.lead} showRole={false} />
      </div>
    </div>
  );
}

function GroupMembers({ group }: { group: OrgGroup }) {
  if (!group.members?.length) return null;

  return (
    <ul className={cn('grid min-w-0 gap-x-8 gap-y-3 sm:grid-cols-2')}>
      {group.members.map((member, memberIndex) => (
        <li
          key={`${group.name}-${member.name}-${member.role ?? member.cohort}-${memberIndex}`}
          className={cn('min-w-0 border-b border-border-subtle pb-3')}
        >
          <PersonLine person={member} />
        </li>
      ))}
    </ul>
  );
}

function GroupSubgroups({ group }: { group: OrgGroup }) {
  if (!group.subgroups?.length) return null;

  return (
    <div className={cn('grid min-w-0 gap-7 sm:grid-cols-2')}>
      {group.subgroups.map((subgroup, subgroupIndex) => (
        <section
          key={`${group.name}-${subgroup.title}-${subgroupIndex}`}
          aria-labelledby={`organization-${group.name}-${subgroupIndex}`}
          className={cn('min-w-0')}
        >
          <h5
            id={`organization-${group.name}-${subgroupIndex}`}
            className={cn(
              'border-b border-border pb-2 font-serif text-sm font-bold leading-6 text-text-primary [overflow-wrap:anywhere]',
            )}
          >
            {subgroup.title}
          </h5>
          <ul className={cn('mt-3 space-y-2.5')}>
            {subgroup.members.map((member, memberIndex) => (
              <li
                key={`${group.name}-${subgroup.title}-${member.name}-${member.cohort}-${memberIndex}`}
                className={cn('min-w-0')}
              >
                <PersonLine person={member} />
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}

function OrganizationGroup({ group, groupIndex }: { group: OrgGroup; groupIndex: number }) {
  return (
    <li className={cn('border-b border-border py-8 md:py-10')}>
      <article className={cn('min-w-0 md:grid md:grid-cols-[180px_minmax(0,1fr)] md:gap-10')}>
        <header className={cn('min-w-0')}>
          <p
            aria-hidden="true"
            className={cn('text-xs font-semibold tracking-[0.18em] text-text-placeholder')}
          >
            {String(groupIndex + 1).padStart(2, '0')}
          </p>
          <h4
            className={cn(
              'mt-2 font-serif text-2xl font-bold leading-tight text-text-primary [overflow-wrap:anywhere]',
            )}
          >
            {group.name}
          </h4>
        </header>

        <div className={cn('mt-6 min-w-0 space-y-7 md:mt-0')}>
          <GroupLead group={group} />
          <GroupMembers group={group} />
          <GroupSubgroups group={group} />
        </div>
      </article>
    </li>
  );
}

export function OrganizationSection() {
  return (
    <section
      id="organization"
      aria-labelledby="organization-heading"
      className={cn(
        'scroll-mt-[var(--landing-header-height)] overflow-x-clip border-t border-border-subtle bg-surface px-5 py-16 sm:px-8 md:px-6 md:py-24',
      )}
    >
      <div className={cn('mx-auto max-w-[1080px]')}>
        <header className={cn('max-w-2xl')}>
          <p
            className={cn(
              'text-xs font-semibold uppercase tracking-[0.24em] text-text-placeholder',
            )}
          >
            Organization
          </p>
          <h2
            id="organization-heading"
            className={cn(
              'mt-3 font-serif text-3xl font-bold leading-tight tracking-tight text-text-primary sm:text-4xl md:text-5xl',
            )}
          >
            조직도
          </h2>
          <p className={cn('mt-5 text-sm leading-7 text-text-secondary sm:text-base')}>
            동문의 뜻을 투명하고 지속 가능한 장학사업으로 연결하는 사람들입니다.
          </p>
        </header>

        <div className={cn('mt-10 sm:mt-12 md:mt-16')}>
          <Card
            variant="elevated"
            padding="none"
            className={cn('mx-auto max-w-xl overflow-hidden')}
          >
            <article className={cn('px-6 py-8 text-center sm:px-10 sm:py-10')}>
              <div
                className={cn(
                  'mx-auto flex size-12 items-center justify-center rounded-full bg-primary-light text-primary',
                )}
              >
                <UserRound aria-hidden="true" className={cn('size-6')} />
              </div>
              <p
                className={cn(
                  'mt-5 text-xs font-semibold uppercase tracking-[0.2em] text-text-tertiary',
                )}
              >
                회장
              </p>
              <h3
                className={cn(
                  'mt-2 font-serif text-2xl font-bold leading-tight text-text-primary sm:text-3xl [overflow-wrap:anywhere]',
                )}
              >
                {ORG_CHAIR.name}
              </h3>
              <p className={cn('mt-2 text-sm leading-6 text-text-tertiary')}>
                {ORG_CHAIR.cohort}
              </p>
              {ORG_CHAIR.role && (
                <p
                  className={cn(
                    'mx-auto mt-3 max-w-md text-sm leading-6 text-text-secondary [overflow-wrap:anywhere]',
                  )}
                >
                  {ORG_CHAIR.role}
                </p>
              )}
            </article>
          </Card>

          <div className={cn('mx-auto h-12 w-px bg-border')} aria-hidden="true" />

          <div className={cn('flex items-center gap-3 border-b border-border pb-5')}>
            <Network aria-hidden="true" className={cn('size-5 shrink-0 text-primary')} />
            <h3 className={cn('font-serif text-lg font-bold text-text-primary sm:text-xl')}>
              주요 조직
            </h3>
          </div>

          <ol className={cn('min-w-0')}>
            {ORG_GROUPS.map((group, groupIndex) => (
              <OrganizationGroup
                key={`${group.name}-${groupIndex}`}
                group={group}
                groupIndex={groupIndex}
              />
            ))}
          </ol>
        </div>
      </div>
    </section>
  );
}
