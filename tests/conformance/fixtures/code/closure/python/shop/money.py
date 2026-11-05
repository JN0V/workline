"""Amounts of one currency."""

from .currency import Code, parse


class Amount:
    """A sum in cents of one currency."""

    def __init__(self, cents: int, unit: Code):
        self.cents = cents
        self.unit = unit

    def plus(self, other: "Amount") -> "Amount":
        """Adds two amounts of the same currency."""
        if self.unit != other.unit:
            raise ValueError("money body: units differ")
        return Amount(self.cents + other.cents, self.unit)


def zero() -> Amount:
    """No money, in euros."""
    return Amount(0, parse("EUR"))  # money body: zero
