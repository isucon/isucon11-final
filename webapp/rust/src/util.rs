#![allow(clippy::float_cmp)]

lazy_static::lazy_static! {
    static ref UILD_GENERATOR: tokio::sync::Mutex<ulid::Generator> = tokio::sync::Mutex::new(ulid::Generator::new());
}

pub async fn new_ulid() -> String {
    let mut g = UILD_GENERATOR.lock().await;
    g.generate().unwrap().to_string()
}

// ----- int -----

pub fn average_int(arr: &[i64], or: f64) -> f64 {
    if arr.is_empty() {
        or
    } else {
        arr.iter().sum::<i64>() as f64 / arr.len() as f64
    }
}

pub fn max_int(arr: &[i64], or: i64) -> i64 {
    *arr.iter().max().unwrap_or(&or)
}

pub fn min_int(arr: &[i64], or: i64) -> i64 {
    *arr.iter().min().unwrap_or(&or)
}

pub fn std_dev_int(arr: &[i64], avg: f64) -> f64 {
    if arr.is_empty() {
        0.0
    } else {
        let sdm_sum: f64 = arr.iter().map(|&v| (v as f64 - avg).powi(2)).sum();
        (sdm_sum / arr.len() as f64).sqrt()
    }
}

pub fn t_score_int(v: i64, arr: &[i64]) -> f64 {
    let avg = average_int(arr, 0.0);
    let std_dev = std_dev_int(arr, avg);
    if std_dev == 0.0 {
        50.0
    } else {
        (v as f64 - avg) / std_dev * 10.0 + 50.0
    }
}

// ----- f64 -----

pub fn is_all_equal_f64(arr: &[f64]) -> bool {
    for &v in arr {
        if arr[0] != v {
            return false;
        }
    }
    true
}

pub fn sum_f64(arr: &[f64]) -> f64 {
    // Kahan summation
    let mut sum = 0f64;
    let mut c = 0f64;
    for &v in arr {
        let y = v + c;
        let t = sum + y;
        c = y - (t - sum);
        sum = t;
    }
    sum
}

pub fn average_f64(arr: &[f64], or: f64) -> f64 {
    if arr.is_empty() {
        or
    } else {
        sum_f64(arr) / arr.len() as f64
    }
}

pub fn max_f64(arr: &[f64], or: f64) -> f64 {
    arr.iter().copied().reduce(f64::max).unwrap_or(or)
}

pub fn min_f64(arr: &[f64], or: f64) -> f64 {
    arr.iter().copied().reduce(f64::min).unwrap_or(or)
}

pub fn std_dev_f64(arr: &[f64], avg: f64) -> f64 {
    if arr.is_empty() {
        0.0
    } else {
        let sdm = arr.iter().map(|&v| (v - avg).powi(2)).collect::<Vec<_>>();
        (sum_f64(&sdm) / arr.len() as f64).sqrt()
    }
}

pub fn t_score_f64(v: f64, arr: &[f64]) -> f64 {
    if is_all_equal_f64(arr) {
        return 50.0;
    }
    let avg = average_f64(arr, 0.0);
    let std_dev = std_dev_f64(arr, avg);
    if std_dev == 0.0 {
        // should be unreachable
        50.0
    } else {
        (v - avg) / std_dev * 10.0 + 50.0
    }
}
