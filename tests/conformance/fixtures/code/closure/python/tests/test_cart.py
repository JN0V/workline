from shop.cart import Cart


def test_empty_cart():
    assert Cart([]).total().cents == 0
