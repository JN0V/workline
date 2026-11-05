package shop

import kotlin.test.Test
import kotlin.test.assertEquals

class CartTest {
    @Test
    fun emptyCartCostsNothing() {
        assertEquals(0, Cart(emptyList()).total().cents)
    }
}
