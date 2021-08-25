#data "aws_ami" "contestant" {
#  owners      = ["self"]
#  most_recent = true
#  name_regex  = "^isucon11f-amd64-contestant-\\d{8}-\\d{4}-[0-9a-f]{40}$"

#  filter {
#    name   = "root-device-type"
#    values = ["ebs"]
#  }

#  filter {
#    name   = "virtualization-type"
#    values = ["hvm"]
#  }
#}

variable "contestant_names" {
  type    = list(string)
  default = ["takonomura", "temma", "hosshii", "buchy", "oribe"]
}

variable "contestant_team_ids" {
  type = map(string)
  default = {
    takonomura = "1"
    temma      = "8"
    hosshii    = "11"
    buchy      = "16"
    oribe      = "19"
  }
}

resource "aws_instance" "contestant-1" {
  for_each = toset(var.contestant_names)

  #ami           = data.aws_ami.contestant.id
  ami           = "ami-02fbcf0d7a6dcbd71"
  instance_type = "c5.large"

  availability_zone = var.availability_zones[0]
  subnet_id         = data.aws_subnet.public[0].id

  vpc_security_group_ids = [
    data.aws_security_group.default.id,
    aws_security_group.final-dev-contestant.id,
  ]

  tags = {
    Name = "final-dev-contestant-${each.key}-1"

    IsuconTeamID      = var.contestant_team_ids[each.key]
    IsuconInstanceNum = "1"
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = "30"
    tags = {
      Name    = "final-dev-contestant-${each.key}-1"
      Project = "final-dev"
    }
  }

  user_data = file("${path.module}/contestant-user-data.sh")
}

resource "aws_eip" "contestant-1" {
  for_each = toset(var.contestant_names)

  vpc      = true
  instance = aws_instance.contestant-1[each.key].id
}
