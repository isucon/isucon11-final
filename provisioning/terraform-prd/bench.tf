data "aws_ami" "bench" {
  owners      = ["self"]
  most_recent = true
  name_regex  = "^isucon11f-amd64-bench-\\d{8}-\\d{4}-[0-9a-f]{40}$"

  filter {
    name   = "root-device-type"
    values = ["ebs"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}

data "aws_ssm_parameter" "bench_token" {
  name = "/hako/isuxportal-prd/ISUXPORTAL_BENCH_TOKEN"
}

resource "aws_instance" "bench" {
  for_each = toset(var.team_ids)

  ami           = data.aws_ami.bench.id
  instance_type = "c5.xlarge"

  subnet_id         = aws_subnet.bench.id
  availability_zone = aws_subnet.bench.availability_zone
  private_ip        = cidrhost(aws_subnet.bench.cidr_block, index(var.team_ids, each.key) + 201)

  vpc_security_group_ids = [
    aws_security_group.bench.id,
  ]

  tags = {
    Name = format("final-prd-bench-%03d", tonumber(each.key))
    Role = "bench"
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = "20"
    tags = {
      Name    = format("final-prd-bench-%03d", tonumber(each.key))
      Project = "final"
    }
  }

  user_data = templatefile("${path.module}/bench-user-data.sh.tpl", { isuxportal_supervisor_token = data.aws_ssm_parameter.bench_token.value, isuxportal_supervisor_team_id = each.key })
}
