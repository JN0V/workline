package shop.money;

import shop.currency.Code;

/** A sum in cents of one currency. */
public record Amount(long cents, Code unit) {
    /** No money, in euros. */
    public static Amount zero() {
        return new Amount(0, Code.parse("EUR")); // money body: zero
    }

    /** Adds two amounts of the same currency. */
    public Amount plus(Amount other) {
        if (!unit.equals(other.unit)) {
            throw new IllegalArgumentException("money body: units differ");
        }
        return new Amount(cents + other.cents, unit);
    }
}
