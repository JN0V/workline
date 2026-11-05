/** Says whether a currency has a rate. */
export function known(code: string): boolean {
  return code === 'EUR' || code === 'USD'; // rates body
}
