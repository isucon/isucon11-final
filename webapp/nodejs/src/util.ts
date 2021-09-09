import { monotonicFactory } from "ulid";

const ulid = monotonicFactory();

export function newUlid(): string {
  return ulid();
}

export function max(arr: number[], or: number): number {
  if (arr.length === 0) {
    return or;
  }
  return Math.max(...arr);
}

export function min(arr: number[], or: number): number {
  if (arr.length === 0) {
    return or;
  }
  return Math.min(...arr);
}

// ----- integer -----

export function averageInt(arr: number[], or: number): number {
  if (arr.length === 0) {
    return or;
  }
  let sum = 0;
  arr.forEach((v) => {
    sum += v;
  });
  return sum / arr.length;
}

function stdDevInt(arr: number[], avg: number): number {
  if (arr.length === 0) {
    return 0;
  }
  let sdmSum = 0;
  arr.forEach((v) => {
    sdmSum += Math.pow(v - avg, 2);
  });
  return Math.sqrt(sdmSum / arr.length);
}

export function tScoreInt(v: number, arr: number[]): number {
  const avg = averageInt(arr, 0);
  const stdDev = stdDevInt(arr, avg);
  if (stdDev === 0) {
    return 50;
  } else {
    return ((v - avg) / stdDev) * 10 + 50;
  }
}

// ----- float -----

function isAllEqualFloat(arr: number[]): boolean {
  return arr.every((v) => arr[0] === v);
}

function sumFloat(arr: number[]): number {
  // Kahan summation
  let sum = 0;
  let c = 0;
  arr.forEach((v) => {
    const y = v + c;
    const t = sum + y;
    c = y - (t - sum);
    sum = t;
  });
  return sum;
}

export function averageFloat(arr: number[], or: number): number {
  if (arr.length === 0) {
    return or;
  }
  return sumFloat(arr) / arr.length;
}

function stdDevFloat(arr: number[], avg: number): number {
  if (arr.length === 0) {
    return 0;
  }
  const sdm = Array.from(Array(arr.length), () => 0);
  arr.forEach((v, i) => {
    sdm[i] = Math.pow(v - avg, 2);
  });
  return Math.sqrt(sumFloat(sdm) / arr.length);
}

export function tScoreFloat(v: number, arr: number[]): number {
  if (isAllEqualFloat(arr)) {
    return 50;
  }
  const avg = averageFloat(arr, 0);
  const stdDev = stdDevFloat(arr, avg);
  if (stdDev === 0) {
    // should be unreachable
    return 50;
  } else {
    return ((v - avg) / stdDev) * 10 + 50;
  }
}
