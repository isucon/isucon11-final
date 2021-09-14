data "aws_vpc" "main" {
  id = "vpc-04f36e0596c6daf7f"
}

data "aws_security_group" "default" {
  vpc_id = data.aws_vpc.main.id
  name   = "default"
}

variable "availability_zones" {
  type    = list(string)
  default = ["ap-northeast-1a", "ap-northeast-1c", "ap-northeast-1d"]
}

data "aws_subnet" "private" {
  count             = 3
  vpc_id            = data.aws_vpc.main.id
  availability_zone = var.availability_zones[count.index]
  filter {
    name   = "tag:Name"
    values = ["${var.availability_zones[count.index]}-private"]
  }
  filter {
    name   = "tag:Tier"
    values = ["private"]
  }
}

data "aws_subnet" "public" {
  count             = 3
  vpc_id            = data.aws_vpc.main.id
  availability_zone = var.availability_zones[count.index]
  filter {
    name   = "tag:Name"
    values = ["${var.availability_zones[count.index]}-public"]
  }
  filter {
    name   = "tag:Tier"
    values = ["public"]
  }
}

resource "aws_security_group" "final-dev-contestant" {
  vpc_id      = data.aws_vpc.main.id
  name        = "final-dev-contestant"
  description = "security group for final-dev contestant instances"
}

resource "aws_security_group" "final-dev-bench" {
  vpc_id      = data.aws_vpc.main.id
  name        = "final-dev-bench"
  description = "security group for final-dev benchmarker instances"
}

resource "aws_security_group_rule" "final-dev-contestant-ingress-ssh" {
  security_group_id = aws_security_group.final-dev-contestant.id
  type              = "ingress"
  protocol          = "tcp"
  from_port         = 22
  to_port           = 22
  cidr_blocks       = ["0.0.0.0/0"]
}

resource "aws_security_group_rule" "final-dev-contestant-ingress-benchmark" {
  security_group_id        = aws_security_group.final-dev-contestant.id
  type                     = "ingress"
  protocol                 = "tcp"
  from_port                = 80
  to_port                  = 80
  source_security_group_id = aws_security_group.final-dev-bench.id
}

resource "aws_security_group_rule" "final-dev-contestant-ingress-benchmark-https" {
  security_group_id        = aws_security_group.final-dev-contestant.id
  type                     = "ingress"
  protocol                 = "tcp"
  from_port                = 443
  to_port                  = 443
  source_security_group_id = aws_security_group.final-dev-bench.id
}

resource "aws_security_group_rule" "final-dev-contestant-ingress-contestant" {
  security_group_id        = aws_security_group.final-dev-contestant.id
  type                     = "ingress"
  protocol                 = "all"
  from_port                = 0
  to_port                  = 0
  source_security_group_id = aws_security_group.final-dev-contestant.id
}

data "aws_security_group" "prometheus" {
  vpc_id = data.aws_vpc.main.id
  name   = "prometheus"
}

resource "aws_security_group_rule" "final-dev-bench-ingress-prometheus" {
  security_group_id        = aws_security_group.final-dev-bench.id
  type                     = "ingress"
  protocol                 = "tcp"
  from_port                = 9100
  to_port                  = 9100
  source_security_group_id = data.aws_security_group.prometheus.id
}
