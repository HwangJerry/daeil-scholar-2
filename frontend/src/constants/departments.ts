// departments — Daeil Foreign Language High School department enum values
export const DEPARTMENTS = [
  '영어과',
  '독일어과',
  '일본어과',
  '중국어과',
  '프랑스어과',
  '스페인어과',
] as const;

export type Department = (typeof DEPARTMENTS)[number];

export function isValidDepartment(value: string): value is Department {
  return (DEPARTMENTS as readonly string[]).includes(value);
}
