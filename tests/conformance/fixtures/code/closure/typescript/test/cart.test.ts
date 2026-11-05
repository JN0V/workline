import { Cart } from '../src/cart';

test('an empty cart costs nothing', () => {
  expect(new Cart([]).total().cents).toBe(0);
});
