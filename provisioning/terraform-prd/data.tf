variable "team_ids" {
  type    = list(string)
  default = [221, 299, 3, 153, 633, 65, 42, 295, 213, 17, 321, 163, 337, 34, 193, 40, 578, 165, 429, 358, 209, 459, 57, 269, 83, 4, 231, 179, 457, 35, 1]
}

variable "checker_tokens" {
  type = map(string)
  default = {
  }
}
