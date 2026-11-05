import { Amount, plus, zero } from './money';

/** The lines of an order. */
export class Cart {
  constructor(public lines: Amount[]) {}

  /** Adds the cart's lines. */
  total(): Amount {
    let sum = zero();
    for (const line of this.lines) {
      sum = plus(sum, line);
    }
    return sum;
  }
}
