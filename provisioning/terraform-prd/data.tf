variable "team_ids" {
  type    = list(string)
  default = []
}

variable "checker_tokens" {
  type = map(string)
  default = {
  }
}
