/** A brace opening a block, as a pattern. */
const open = /\{/;

/** Splits a text into its tokens. */
export function tokens(text: string): string[] {
  return text.split(open); // lexer body: split
}

/** Says whether a token is a word. */
export function isWord(token: string): boolean {
  return /^\w+$/.test(token); // lexer body: word
}
