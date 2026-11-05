import { known } from './rates';

/** A currency's ISO code. */
export type Code = string;

/** Reads a currency's code. */
export function parse(text: string): Code {
  if (!known(text)) {
    throw new Error('currency body: unknown ' + text);
  }
  return text;
}
