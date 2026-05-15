// departments — Daeil Foreign Language High School department enum values
export const DEPARTMENTS = [
  '프랑스어',
  '독일어',
  '일본어',
  '중국어',
  '스페인어',
  '러시아어',
  '영어',
] as const;

export type Department = (typeof DEPARTMENTS)[number];

export function isValidDepartment(value: string): value is Department {
  return (DEPARTMENTS as readonly string[]).includes(value);
}
