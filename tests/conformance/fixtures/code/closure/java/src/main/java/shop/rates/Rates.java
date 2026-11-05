package shop.rates;

/** The currencies with a rate. */
public final class Rates {
    public static boolean known(String code) {
        return code.equals("EUR") || code.equals("USD"); // rates body
    }
}
