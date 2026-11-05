package shop;

import java.util.List;
import shop.money.Amount;

/** The lines of an order. */
public class Cart {
    private final List<Amount> lines;

    public Cart(List<Amount> lines) {
        this.lines = lines;
    }

    /** Adds the cart's lines. */
    public Amount total() {
        Amount sum = Amount.zero();
        for (Amount line : lines) {
            sum = sum.plus(line);
        }
        return sum;
    }
}
