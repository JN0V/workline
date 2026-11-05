import { tokens } from './lexer';

/** Reads a cart's lines from its text. */
export function lines(text: string): string[] {
  return tokens(text).filter((t) => t !== '{');
}
