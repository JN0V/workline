import { Code, parse } from './currency';

/** A sum in cents of one currency. */
export interface Amount {
  cents: number;
  unit: Code;
}

/** No money, in euros. */
export function zero(): Amount {
  return { cents: 0, unit: parse('EUR') }; // money body: zero
}

/** Adds two amounts of the same currency. */
export function plus(a: Amount, b: Amount): Amount {
  if (a.unit !== b.unit) {
    throw new Error('money body: units differ');
  }
  return { cents: a.cents + b.cents, unit: a.unit };
}
