// OrganizationPage — Foundation org chart: chair, board, audit, secretariat, external relations, PR
import { Card } from '../components/ui/Card';
import { InfoPageShell } from '../components/info/InfoPageShell';
import { ORG_CHAIR, ORG_GROUPS, type OrgPerson } from '../constants/aboutContent';

function PersonRow({ person, showRole = true }: { person: OrgPerson; showRole?: boolean }) {
  return (
    <div className="flex flex-wrap items-baseline gap-x-2 gap-y-0.5 text-body-sm">
      <span className="font-serif font-semibold text-text-primary">{person.name}</span>
      <span className="text-text-tertiary">({person.cohort})</span>
      {showRole && person.role && (
        <span className="text-text-secondary">— {person.role}</span>
      )}
    </div>
  );
}

export function OrganizationPage() {
  return (
    <InfoPageShell
      title="조직도"
      subtitle="본 장학회는 아래와 같이 구성되어 있습니다."
      canonicalPath="/organization"
    >
      <Card variant="elevated" padding="lg" className="space-y-2">
        <p className="text-2xs uppercase tracking-wider text-text-tertiary">회장</p>
        <div className="space-y-1">
          <p className="font-serif text-body-md font-semibold text-text-primary">
            {ORG_CHAIR.name} <span className="text-text-tertiary">({ORG_CHAIR.cohort})</span>
          </p>
          {ORG_CHAIR.role && (
            <p className="text-body-sm text-text-secondary">{ORG_CHAIR.role}</p>
          )}
        </div>
      </Card>

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
        {ORG_GROUPS.map((group) => (
          <Card
            key={group.name}
            variant="default"
            padding="lg"
            className="space-y-4"
          >
            <h2 className="font-serif text-body-md font-semibold text-text-primary">
              {group.name}
            </h2>

            {group.lead && (
              <div className="space-y-1 border-l border-border pl-3">
                <p className="text-2xs uppercase tracking-wider text-text-tertiary">
                  {group.lead.role ?? '대표'}
                </p>
                <PersonRow person={group.lead} showRole={false} />
              </div>
            )}

            {group.members && group.members.length > 0 && (
              <ul className="space-y-2 border-l border-border pl-3">
                {group.members.map((member) => (
                  <li key={`${group.name}-${member.name}-${member.role ?? ''}`}>
                    <PersonRow person={member} />
                  </li>
                ))}
              </ul>
            )}

            {group.subgroups && group.subgroups.length > 0 && (
              <div className="space-y-3 border-l border-border pl-3">
                {group.subgroups.map((subgroup) => (
                  <div key={`${group.name}-${subgroup.title}`} className="space-y-1.5">
                    <p className="text-body-sm font-semibold text-text-primary">
                      {subgroup.title}
                    </p>
                    <ul className="space-y-1">
                      {subgroup.members.map((member) => (
                        <li key={`${subgroup.title}-${member.name}`}>
                          <PersonRow person={member} />
                        </li>
                      ))}
                    </ul>
                  </div>
                ))}
              </div>
            )}
          </Card>
        ))}
      </div>
    </InfoPageShell>
  );
}
