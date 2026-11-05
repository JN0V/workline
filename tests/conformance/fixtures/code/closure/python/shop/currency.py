"""The currencies the shop takes."""

from .rates import known

Code = str


def parse(text: str) -> Code:
    """Reads a currency's code."""
    if not known(text):
        raise ValueError("currency body: unknown " + text)
    return text
