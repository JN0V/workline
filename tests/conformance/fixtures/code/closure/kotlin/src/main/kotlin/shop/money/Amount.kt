package shop.money

import shop.currency.Code

/** A sum in cents of one currency. */
data class Amount(val cents: Long, val unit: Code) : Comparable<Amount> {
    /** Adds two amounts of the same currency. */
    fun plus(other: Amount): Amount {
        require(unit == other.unit) { "money body: units differ" }
        return Amount(cents + other.cents, unit)
    }

    override fun compareTo(other: Amount): Int {
        return cents.compareTo(other.cents) // money body: compare
    }

    companion object {
        /** No money, in euros. */
        fun zero(): Amount {
            return Amount(0, Code.parse("EUR")) // money body: zero
        }
    }
}
