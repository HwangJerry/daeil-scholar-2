// notificationTemplateText — pure text measurements and placeholder handling for notification templates
import type { NotificationTemplatePlaceholder } from '../types/notificationTemplate.ts';

/** Mirrors the server's `templateBraceTokenPattern`; malformed tokens included. */
const BRACE_TOKEN_PATTERN = /\{[^{}]*\}/g;
/** Mirrors the server's `templatePlaceholderNamePattern`. */
const PLACEHOLDER_NAME_PATTERN = /^[a-zA-Z][a-zA-Z0-9]*$/;
const ASCII_MAX_CODE_POINT = 0x7f;
const ASCII_BYTES = 1;
const WIDE_BYTES = 2;

/**
 * Length of the text once the SMS gateway encodes it as EUC-KR: ASCII costs one
 * byte and every other character two. The server is authoritative — this is the
 * counter the administrator watches while typing.
 */
export function eucKrByteLength(text: string): number {
  return Array.from(text).reduce((bytes, character) => {
    const codePoint = character.codePointAt(0) ?? 0;
    return bytes + (codePoint <= ASCII_MAX_CODE_POINT ? ASCII_BYTES : WIDE_BYTES);
  }, 0);
}

/** Number of characters as the push payload counts them (code points, not UTF-16 units). */
export function runeLength(text: string): number {
  return Array.from(text).length;
}

/** Every `{...}` token in the text, braces included. */
export function braceTokens(text: string): string[] {
  return text.match(BRACE_TOKEN_PATTERN) ?? [];
}

/** Strips the braces from a token returned by `braceTokens`. */
export function braceTokenName(token: string): string {
  return token.slice(1, -1);
}

export function containsPlaceholder(text: string, name: string): boolean {
  return text.includes(`{${name}}`);
}

/**
 * Tokens the template may not use: a misspelling such as `{cod}` would otherwise
 * reach a member verbatim in place of their code.
 */
export function unknownBraceTokens(
  text: string,
  placeholders: NotificationTemplatePlaceholder[],
): string[] {
  const allowed = new Set(placeholders.map((placeholder) => placeholder.name));
  const unknown = braceTokens(text).filter((token) => {
    const name = braceTokenName(token);
    return !PLACEHOLDER_NAME_PATTERN.test(name) || !allowed.has(name);
  });
  return Array.from(new Set(unknown));
}

/**
 * Reports a `{` or `}` that is not part of a `{token}`, such as the outer braces
 * of `{{code}}`. Those braces are not substituted, so they would be delivered to
 * the member literally.
 */
export function hasStrayBrace(text: string): boolean {
  return /[{}]/.test(text.replace(BRACE_TOKEN_PATTERN, ''));
}

/** Required placeholders the body is missing. */
export function missingRequiredPlaceholders(
  body: string,
  placeholders: NotificationTemplatePlaceholder[],
): NotificationTemplatePlaceholder[] {
  return placeholders.filter(
    (placeholder) => placeholder.required && !containsPlaceholder(body, placeholder.name),
  );
}

/**
 * The text as a member would receive it: every allowed placeholder replaced by
 * its worst-case sample. This is both the preview and what the SMS byte counter
 * measures, matching the server's `renderTemplateWithSamples`.
 */
export function renderWithSamples(
  text: string,
  placeholders: NotificationTemplatePlaceholder[],
): string {
  const samples = new Map(placeholders.map((placeholder) => [placeholder.name, placeholder.sample]));
  // One pass over the text, like the server's strings.NewReplacer: a sample that
  // itself contains `{...}` is output as-is rather than substituted again.
  return text.replace(BRACE_TOKEN_PATTERN, (token) => samples.get(braceTokenName(token)) ?? token);
}

/** Inserts `{name}` into `text` at the selection, returning the text and the new caret. */
export function insertPlaceholder(
  text: string,
  name: string,
  selectionStart: number,
  selectionEnd: number,
): { text: string; caret: number } {
  const token = `{${name}}`;
  const start = Math.min(Math.max(selectionStart, 0), text.length);
  const end = Math.min(Math.max(selectionEnd, start), text.length);
  return {
    text: `${text.slice(0, start)}${token}${text.slice(end)}`,
    caret: start + token.length,
  };
}
