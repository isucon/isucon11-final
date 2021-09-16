<?php

declare(strict_types=1);

namespace util;

use Ulid\Ulid;

function newUlid(): string
{
    return (string)Ulid::generate();
}

// ----- common for int and float

/**
 * @param array<int>|array<float> $arr
 */
function average(array $arr, float $or): float
{
    if (count($arr) === 0) {
        return $or;
    }

    return (float)array_sum($arr) / count($arr);
}

/**
 * @param array<int>|array<float> $arr
 */
function max(array $arr, int|float $or): int|float
{
    if (count($arr) === 0) {
        return $or;
    }

    return \max($arr);
}

/**
 * @param array<int>|array<float> $arr
 */
function min(array $arr, int|float $or): int|float
{
    if (count($arr) === 0) {
        return $or;
    }

    return \min($arr);
}

/**
 * @param array<int>|array<float> $arr
 */
function stdDev(array $arr, float $avg): float
{
    if (count($arr) === 0) {
        return 0;
    }

    $sdm = [];
    foreach ($arr as $v) {
        $sdm[] = pow($v - $avg, 2);
    }

    return sqrt(sumFloat($sdm) / count($arr));
}

// ----- int -----

/**
 * @param array<int> $arr
 */
function tScoreInt(int $v, array $arr): float
{
    $avg = average($arr, 0);
    $stdDev = stdDev($arr, $avg);
    if ($stdDev == 0) {
        return 50;
    }

    return ($v - $avg) / $stdDev * 10 + 50;
}

// ----- float -----

/**
 * @param array<float> $arr
 */
function isAllEqualFloat(array $arr): bool
{
    foreach ($arr as $v) {
        if ($arr[0] !== $v) {
            return false;
        }
    }

    return true;
}

/**
 * @param array<float> $arr
 */
function sumFloat(array $arr): float
{
    // Kahan summation
    $sum = 0.0;
    $c = 0.0;
    foreach ($arr as $v) {
        $y = $v + $c;
        $t = $sum + $y;
        $c = $y - ($t - $sum);
        $sum = $t;
    }

    return $sum;
}

/**
 * @param array<float> $arr
 */
function tScoreFloat(float $v, array $arr): float
{
    if (isAllEqualFloat($arr)) {
        return 50;
    }

    $avg = average($arr, 0);
    $stdDev = stdDev($arr, $avg);
    if ($stdDev == 0) {
        // should be unreachable
        return 50;
    }

    return ($v - $avg) / $stdDev * 10 + 50;
}
