variable "team_ids" {
  type    = list(string)
  default = [1, 2, 3, 4]
}

variable "checker_tokens" {
  type = map(string)
  default = {
    1 = "Owx2ylaMlQXOPUsGPuxIpULfPKo_GTn5LvTPK16Vc8XGyik4rPgeAVhMfpK7ao15:eyJ0ZWFtX2lkIjoxLCJleHBpcnkiOjE2MzA4MzYwMDB9"
    2 = "vaJv8XrUfJv-c8h74LND4dEZaoKSUXuiBOrayfw0_pWkQFw-GWcIP2Xh2xe3UAoU:eyJ0ZWFtX2lkIjoyLCJleHBpcnkiOjE2MzA4MzYwMDB9"
    3 = "cfncxaaEL1oviJjG5jLwS6Fntq70ZEFT7ZyDaoi8DXXXQ87K2etAAFrX-sGj_7Gt:eyJ0ZWFtX2lkIjozLCJleHBpcnkiOjE2MzA4MzYwMDB9"
    4 = "NOX-UkAz4NIYrplHaJ0zqfUwTPmjgbyfvqrGu37Ts1rwyAsRxaiPqCs2mxJ7QVBJ:eyJ0ZWFtX2lkIjo0LCJleHBpcnkiOjE2MzA4MzYwMDB9"
  }
}
