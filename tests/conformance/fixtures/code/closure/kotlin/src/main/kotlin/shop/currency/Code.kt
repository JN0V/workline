package shop.currency

import shop.rates.Rates

/** A currency's ISO code. */
class Code private constructor(val iso: String) {
    companion object {
        /** Reads a currency's code. */
        fun parse(text: String): Code {
            require(Rates.known(text)) { "currency body: unknown $text" }
            return Code(text)
        }
    }
}
