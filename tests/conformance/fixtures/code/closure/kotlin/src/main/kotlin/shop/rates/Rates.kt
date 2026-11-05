package shop.rates

/** The currencies with a rate. */
object Rates {
    fun known(code: String): Boolean {
        return code == "EUR" || code == "USD" // rates body
    }
}
