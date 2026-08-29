// OrganizationSection.test — Landing organization hierarchy and shared-content contracts
import { render, screen, within } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { OrganizationSection } from '../components/landing/OrganizationSection';
import { ORG_CHAIR, ORG_GROUPS, type OrgPerson } from '../constants/aboutContent';

function getOrganizationPeople() {
  return ORG_GROUPS.flatMap((group) => [
    ...(group.lead ? [group.lead] : []),
    ...(group.members ?? []),
    ...(group.subgroups?.flatMap((subgroup) => subgroup.members) ?? []),
  ]);
}

function countPeopleByName(people: readonly OrgPerson[]) {
  return people.reduce<Map<string, number>>((nameCounts, person) => {
    nameCounts.set(person.name, (nameCounts.get(person.name) ?? 0) + 1);
    return nameCounts;
  }, new Map());
}

describe('OrganizationSection', () => {
  it('renders the chair before every shared organization group', () => {
    render(<OrganizationSection />);

    const section = document.getElementById('organization');
    expect(section).toHaveClass('scroll-mt-[var(--landing-header-height)]', 'overflow-x-clip');
    expect(section).toHaveAttribute('aria-labelledby', 'organization-heading');

    const chairHeading = screen.getByRole('heading', { name: ORG_CHAIR.name });
    expect(screen.getByText(ORG_CHAIR.cohort)).toBeInTheDocument();
    expect(screen.getByText(ORG_CHAIR.role ?? '')).toBeInTheDocument();
    const groupList = section?.querySelector('ol');
    expect(groupList).not.toBeNull();
    const chairPrecedesGroups = Boolean(
      chairHeading.compareDocumentPosition(groupList as HTMLOListElement) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    );
    expect(chairPrecedesGroups).toBe(true);

    for (const group of ORG_GROUPS) {
      expect(screen.getByRole('heading', { name: group.name })).toBeInTheDocument();
    }
  });

  it('renders every shared lead, member, and subgroup without truncating text', () => {
    render(<OrganizationSection />);

    const people = getOrganizationPeople();
    const peopleByName = countPeopleByName(people);

    for (const [name, expectedCount] of peopleByName) {
      const nameElements = screen.getAllByText(name);
      expect(nameElements).toHaveLength(expectedCount);

      for (const nameElement of nameElements) {
        expect(nameElement.closest('p')).toHaveClass(
          'min-w-0',
          'flex-wrap',
          '[overflow-wrap:anywhere]',
        );
      }
    }

    for (const group of ORG_GROUPS) {
      for (const subgroup of group.subgroups ?? []) {
        const subgroupHeading = screen.getByRole('heading', { name: subgroup.title });
        const subgroupSection = subgroupHeading.closest('section');
        expect(subgroupSection).not.toBeNull();

        for (const member of subgroup.members) {
          expect(within(subgroupSection as HTMLElement).getByText(member.name)).toBeInTheDocument();
        }
      }
    }

    expect(document.querySelectorAll('[class*="line-clamp"], [class*="truncate"]')).toHaveLength(0);
  });
});
