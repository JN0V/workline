package shop.currency;

import shop.rates.Rates;

/** A currency's ISO code. */
public final class Code {
    private final String iso;

    private Code(String iso) {
        this.iso = iso;
    }

    /** Reads a currency's code. */
    public static Code parse(String text) {
        if (!Rates.known(text)) {
            throw new IllegalArgumentException("currency body: unknown " + text);
        }
        return new Code(text);
    }
}
