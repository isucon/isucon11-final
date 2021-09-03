variable "team_ids" {
  type    = list(string)
  default = [1]
}

variable "checker_tokens" {
  type = map(string)
  default = {
    1 = "dummy"
  }
}
