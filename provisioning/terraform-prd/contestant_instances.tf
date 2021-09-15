data "aws_ami" "contestant" {
  owners      = ["self"]
  most_recent = true
  name_regex  = "^isucon11f-amd64-contestant-\\d{8}-\\d{4}-[0-9a-f]{40}$"

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

resource "aws_instance" "contestant-1" {
  for_each = toset(var.team_ids)

  ami           = data.aws_ami.contestant.id
  instance_type = "c5.large"

  subnet_id         = aws_subnet.contestant[each.key].id
  availability_zone = aws_subnet.contestant[each.key].availability_zone
  private_ip        = cidrhost(aws_subnet.contestant[each.key].cidr_block, 101)

  vpc_security_group_ids = [
    aws_security_group.contestant[each.key].id,
  ]


  tags = {
    Name = format("final-prd-contestant-%02d-1", tonumber(each.key))
    Role = "contestant"

    IsuconTeamID      = each.key
    IsuconInstanceNum = "1"
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = "30"
    tags = {
      Name    = format("final-prd-contestant-%02d-1", tonumber(each.key))
      Project = "final-prd"
    }
  }

  user_data = templatefile("${path.module}/contestant-user-data.sh.tpl", { checker_token = var.checker_tokens[each.key] })
}

resource "aws_eip" "contestant-1" {
  for_each = toset(var.team_ids)

  vpc      = true
  instance = aws_instance.contestant-1[each.key].id

  tags = {
    Name = format("final-prd-contestant-%02d-1", tonumber(each.key))
  }
}

resource "aws_instance" "contestant-2" {
  for_each = toset(var.team_ids)

  ami           = data.aws_ami.contestant.id
  instance_type = "c5.large"

  subnet_id         = aws_subnet.contestant[each.key].id
  availability_zone = aws_subnet.contestant[each.key].availability_zone
  private_ip        = cidrhost(aws_subnet.contestant[each.key].cidr_block, 102)

  vpc_security_group_ids = [
    aws_security_group.contestant[each.key].id,
  ]


  tags = {
    Name = format("final-prd-contestant-%02d-2", tonumber(each.key))
    Role = "contestant"

    IsuconTeamID      = each.key
    IsuconInstanceNum = "2"
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = "30"
    tags = {
      Name    = format("final-prd-contestant-%02d-2", tonumber(each.key))
      Project = "final-prd"
    }
  }

  user_data = templatefile("${path.module}/contestant-user-data.sh.tpl", { checker_token = var.checker_tokens[each.key] })
}

resource "aws_eip" "contestant-2" {
  for_each = toset(var.team_ids)

  vpc      = true
  instance = aws_instance.contestant-2[each.key].id

  tags = {
    Name = format("final-prd-contestant-%02d-2", tonumber(each.key))
  }
}

resource "aws_instance" "contestant-3" {
  for_each = toset(var.team_ids)

  ami           = data.aws_ami.contestant.id
  instance_type = "c5.large"

  subnet_id         = aws_subnet.contestant[each.key].id
  availability_zone = aws_subnet.contestant[each.key].availability_zone
  private_ip        = cidrhost(aws_subnet.contestant[each.key].cidr_block, 103)

  vpc_security_group_ids = [
    aws_security_group.contestant[each.key].id,
  ]


  tags = {
    Name = format("final-prd-contestant-%02d-3", tonumber(each.key))
    Role = "contestant"

    IsuconTeamID      = each.key
    IsuconInstanceNum = "3"
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = "30"
    tags = {
      Name    = format("final-prd-contestant-%02d-3", tonumber(each.key))
      Project = "final-prd"
    }
  }

  user_data = templatefile("${path.module}/contestant-user-data.sh.tpl", { checker_token = var.checker_tokens[each.key] })
}

resource "aws_eip" "contestant-3" {
  for_each = toset(var.team_ids)

  vpc      = true
  instance = aws_instance.contestant-3[each.key].id

  tags = {
    Name = format("final-prd-contestant-%02d-3", tonumber(each.key))
  }
}
