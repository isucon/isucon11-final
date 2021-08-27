#data "aws_ami" "bench" {
#  owners      = ["self"]
#  most_recent = true
#  name_regex  = "^isucon11f-amd64-bench-\\d{8}-\\d{4}-[0-9a-f]{40}$"

#  filter {
#    name   = "root-device-type"
#    values = ["ebs"]
#  }

#  filter {
#    name   = "virtualization-type"
#    values = ["hvm"]
#  }
#}

data "aws_ssm_parameter" "bench_token" {
  name = "/hako/isuxportal-dev/ISUXPORTAL_BENCH_TOKEN"
}

resource "aws_instance" "bench" {
  count = 1

  #ami           = data.aws_ami.bench.id
  ami           = "ami-01208ab070571fa58"
  instance_type = "c5.large"

  availability_zone = var.availability_zones[0]
  subnet_id         = data.aws_subnet.private[0].id

  vpc_security_group_ids = [
    data.aws_security_group.default.id,
    aws_security_group.final-dev-bench.id,
  ]

  tags = {
    Name = format("final-dev-bench-%02d", count.index + 1)
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = "20"
    tags = {
      Name    = format("final-dev-bench-%02d", count.index + 1)
      Project = "final-dev"
    }
  }

  user_data = templatefile("${path.module}/bench-user-data.sh.tpl", { isuxportal_supervisor_token = data.aws_ssm_parameter.bench_token.value })
}

resource "aws_instance" "bench-test" {
  count = 1

  #ami           = data.aws_ami.bench.id
  ami           = "ami-01208ab070571fa58"
  instance_type = "c5.large"

  availability_zone = var.availability_zones[0]
  subnet_id         = data.aws_subnet.public[0].id

  vpc_security_group_ids = [
    data.aws_security_group.default.id,
    aws_security_group.final-dev-contestant.id,
  ]

  tags = {
    Name = format("final-dev-bench-test-%02d", count.index + 1)
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = "20"
    tags = {
      Name    = format("final-dev-bench-test-%02d", count.index + 1)
      Project = "final-dev"
    }
  }

  user_data = file("${path.module}/bench-test-user-data.sh")
}

resource "aws_eip" "bench-test" {
  count = 1

  vpc      = true
  instance = aws_instance.bench-test[count.index].id
}
