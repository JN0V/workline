"""The currencies with a rate."""


def known(code: str) -> bool:
    """Says whether a currency has a rate."""
    return code in ("EUR", "USD")  # rates body
