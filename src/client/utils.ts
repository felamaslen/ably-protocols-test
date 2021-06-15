export const calculateExponentialDelay = (numRetries: number): number =>
  1000 * 2 ** numRetries;
