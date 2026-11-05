import { lines } from '../src/cart';

test('an empty text has no lines', () => {
  expect(lines('')).toEqual(['']);
});
