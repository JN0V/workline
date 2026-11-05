package shop

import shop.money.Amount

/** The lines of an order. */
class Cart(private val lines: List<Amount>) {
    /** Adds the cart's lines. */
    fun total(): Amount {
        var sum = Amount.zero()
        for (line in lines) {
            sum = sum.plus(line)
        }
        return sum
    }
}
