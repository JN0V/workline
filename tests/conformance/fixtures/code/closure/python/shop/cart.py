"""An order's lines."""

from shop.money import Amount, zero


class Cart:
    """The lines of an order."""

    def __init__(self, lines: list[Amount]):
        self.lines = lines

    def total(self) -> Amount:
        """Adds the cart's lines."""
        total = zero()
        for line in self.lines:
            total = total.plus(line)
        return total
