package shop;

import static org.junit.jupiter.api.Assertions.assertEquals;

import java.util.List;
import org.junit.jupiter.api.Test;

class CartTest {
    @Test
    void emptyCartCostsNothing() {
        assertEquals(0, new Cart(List.of()).total().cents());
    }
}
