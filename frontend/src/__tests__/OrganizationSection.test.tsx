// OrganizationSection.test — Landing organization hierarchy across desktop tree and mobile accordion
import { render, screen, within } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { OrganizationSection } from '../components/landing/OrganizationSection';
import { ORG_CHAIR, ORG_GROUPS, type OrgPerson } from '../constants/aboutContent';

function mockViewport(isMobile: boolean) {
  vi.stubGlobal(
    'matchMedia',
    vi.fn().mockImplementation((query: string) => ({
      matches: query === '(max-width: 767px)' && isMobile,
      media: query,
      onchange: null,
      addEventListener: vi.fn(),
      removeEventListener: vi.fn(),
    })),
  );
}

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
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it('renders the chair before every organization group on desktop', () => {
    mockViewport(false);
    render(<OrganizationSection />);

    const section = document.getElementById('organization');
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

  it('features the board co-lead beside the lead and keeps the full roster intact (desktop)', () => {
    mockViewport(false);
    render(<OrganizationSection />);

    const board = ORG_GROUPS.find((group) => group.name === '이사회');
    expect(board?.coLead).toBeDefined();
    const leadLine = screen
      .getAllByText(board!.lead!.name)
      .map((el) => el.closest('p'))
      .find((p) => p?.textContent === `${board!.lead!.name} · ${board!.lead!.role}`);
    const coLeadLine = screen
      .getAllByText(board!.coLead!.name)
      .map((el) => el.closest('p'))
      .find((p) => p?.textContent === `${board!.coLead!.name} · ${board!.coLead!.role}`);
    expect(leadLine).toBeTruthy();
    expect(coLeadLine).toBeTruthy();

    const people = getOrganizationPeople();
    const peopleByName = countPeopleByName(people);

    for (const [name, expectedCount] of peopleByName) {
      expect(screen.getAllByText(name)).toHaveLength(expectedCount);
    }

    for (const group of ORG_GROUPS) {
      for (const subgroup of group.subgroups ?? []) {
        const subgroupHeading = screen.getByRole('heading', { name: subgroup.title });
        const subgroupSection = subgroupHeading.closest('section');
        expect(subgroupSection).not.toBeNull();

        for (const member of subgroup.members) {
          expect(
            within(subgroupSection as HTMLElement).getByText(member.name),
          ).toBeInTheDocument();
        }
      }
    }

    expect(document.querySelectorAll('[class*="line-clamp"], [class*="truncate"]')).toHaveLength(
      0,
    );
  });

  it('renders the mobile accordion collapsed by default and expands sub-teams on tap', async () => {
    mockViewport(true);
    const user = userEvent.setup();
    render(<OrganizationSection />);

    expect(screen.getByRole('heading', { name: ORG_CHAIR.name })).toBeInTheDocument();
    for (const group of ORG_GROUPS) {
      expect(screen.getByRole('heading', { name: group.name })).toBeInTheDocument();
    }

    expect(screen.queryByText('신보아')).not.toBeInTheDocument();

    const secretariatToggle = screen.getByRole('button', { name: /사무국/ });
    expect(secretariatToggle).toHaveAttribute('aria-expanded', 'false');

    await user.click(secretariatToggle);

    expect(secretariatToggle).toHaveAttribute('aria-expanded', 'true');
    expect(screen.getByText('신보아')).toBeInTheDocument();
  });
});
