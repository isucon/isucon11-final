resource "aws_security_group" "bench" {
  vpc_id      = aws_vpc.main.id
  name        = "final-pre-bench"
  description = "security group for final-pre benchmarker instances"
}

data "aws_security_group" "bastion" {
  vpc_id = data.aws_vpc.portal.id
  name   = "bastion"
}

resource "aws_security_group_rule" "bench-ingress-ssh" {
  security_group_id        = aws_security_group.bench.id
  type                     = "ingress"
  protocol                 = "tcp"
  from_port                = 22
  to_port                  = 22
  source_security_group_id = data.aws_security_group.bastion.id
}

resource "aws_security_group_rule" "bench-egress-all" {
  security_group_id = aws_security_group.bench.id
  type              = "egress"
  protocol          = "all"
  from_port         = 0
  to_port           = 0
  cidr_blocks       = ["0.0.0.0/0"]
}

resource "aws_security_group" "contestant" {
  for_each    = toset(var.team_ids)
  vpc_id      = aws_vpc.main.id
  name        = format("final-pre-contestant-%02d", tonumber(each.key))
  description = "security group for final-pre team #${each.key} contestant instances"
}

resource "aws_security_group_rule" "contestant-ingress-ssh" {
  for_each = toset(var.team_ids)

  security_group_id = aws_security_group.contestant[each.key].id
  type              = "ingress"
  protocol          = "tcp"
  from_port         = 22
  to_port           = 22
  cidr_blocks       = ["0.0.0.0/0"]
}

resource "aws_security_group_rule" "contestant-ingress-benchmark" {
  for_each = toset(var.team_ids)

  security_group_id        = aws_security_group.contestant[each.key].id
  type                     = "ingress"
  protocol                 = "tcp"
  from_port                = 80
  to_port                  = 80
  source_security_group_id = aws_security_group.bench.id
}

resource "aws_security_group_rule" "contestant-ingress-contestant" {
  for_each = toset(var.team_ids)

  security_group_id        = aws_security_group.contestant[each.key].id
  type                     = "ingress"
  protocol                 = "all"
  from_port                = 0
  to_port                  = 0
  source_security_group_id = aws_security_group.contestant[each.key].id
}

resource "aws_security_group_rule" "contestant-egress-all" {
  for_each = toset(var.team_ids)

  security_group_id = aws_security_group.contestant[each.key].id
  type              = "egress"
  protocol          = "all"
  from_port         = 0
  to_port           = 0
  cidr_blocks       = ["0.0.0.0/0"]
}
