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
  default = ["takonomura", "temma", "hosshii", "buchy", "oribe", "eiya", "kanata", "hattori", "takahashi", "eagletmt", "sapphi_red", "sorah", "sapphi_red2", "rust", "ruby", "php", "nodejs", "test1", "test2", "test3"]
}

variable "two_instance_contestant_names" {
  type    = list(string)
  default = ["sapphi_red", "temma", "sorah", "sapphi_red2", "takonomura", "hattori", "ruby", "rust", "php", "nodejs", "test1", "test2", "test3"]
}

variable "three_instance_contestant_names" {
  type    = list(string)
  default = ["sorah", "sapphi_red", "sapphi_red2", "takonomura", "ruby", "test3", "test1"]
}

variable "contestant_team_ids" {
  type = map(string)
  default = {
    takonomura  = "1"
    temma       = "8"
    hosshii     = "11"
    buchy       = "16"
    oribe       = "19"
    eiya        = "13"
    kanata      = "7"
    hattori     = "18"
    takahashi   = "20"
    eagletmt    = "-14"
    sapphi_red  = "-2"
    sorah       = "-15"
    sapphi_red2 = "2"
    rust        = "14"
    ruby        = "15"
    php         = "22"
    nodejs      = "23"
    test1       = "24"
    test2       = "25"
    test3       = "26"
  }
}

resource "aws_instance" "contestant-1" {
  for_each = toset(var.contestant_names)

  #ami           = data.aws_ami.contestant.id
  ami           = "ami-0513f581cefd37db4"
  instance_type = "c5.large"

  availability_zone = var.availability_zones[0]
  subnet_id         = data.aws_subnet.public[0].id

  vpc_security_group_ids = [
    data.aws_security_group.default.id,
    aws_security_group.final-dev-contestant.id,
  ]

  tags = {
    Name = "final-dev-contestant-${each.key}-1"
    Role = "contestant"

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

  lifecycle {
    ignore_changes = [
      ami,
    ]
  }
}

resource "aws_eip" "contestant-1" {
  for_each = toset(var.contestant_names)

  vpc      = true
  instance = aws_instance.contestant-1[each.key].id
}

resource "aws_instance" "contestant-2" {
  for_each = toset(var.two_instance_contestant_names)

  #ami           = data.aws_ami.contestant.id
  ami           = "ami-0513f581cefd37db4"
  instance_type = "c5.large"

  availability_zone = var.availability_zones[0]
  subnet_id         = data.aws_subnet.public[0].id

  vpc_security_group_ids = [
    data.aws_security_group.default.id,
    aws_security_group.final-dev-contestant.id,
  ]

  tags = {
    Name = "final-dev-contestant-${each.key}-2"
    Role = "contestant"

    IsuconTeamID      = var.contestant_team_ids[each.key]
    IsuconInstanceNum = "2"
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = "30"
    tags = {
      Name    = "final-dev-contestant-${each.key}-2"
      Project = "final-dev"
    }
  }

  user_data = file("${path.module}/contestant-user-data.sh")

  lifecycle {
    ignore_changes = [
      ami,
    ]
  }
}

resource "aws_eip" "contestant-2" {
  for_each = toset(var.two_instance_contestant_names)

  vpc      = true
  instance = aws_instance.contestant-2[each.key].id
}

resource "aws_instance" "contestant-3" {
  for_each = toset(var.three_instance_contestant_names)

  #ami           = data.aws_ami.contestant.id
  ami           = "ami-0513f581cefd37db4"
  instance_type = "c5.large"

  availability_zone = var.availability_zones[0]
  subnet_id         = data.aws_subnet.public[0].id

  vpc_security_group_ids = [
    data.aws_security_group.default.id,
    aws_security_group.final-dev-contestant.id,
  ]

  tags = {
    Name = "final-dev-contestant-${each.key}-3"
    Role = "contestant"

    IsuconTeamID      = var.contestant_team_ids[each.key]
    IsuconInstanceNum = "3"
  }

  root_block_device {
    volume_type = "gp3"
    volume_size = "30"
    tags = {
      Name    = "final-dev-contestant-${each.key}-3"
      Project = "final-dev"
    }
  }

  user_data = file("${path.module}/contestant-user-data.sh")

  lifecycle {
    ignore_changes = [
      ami,
    ]
  }
}

resource "aws_eip" "contestant-3" {
  for_each = toset(var.three_instance_contestant_names)

  vpc      = true
  instance = aws_instance.contestant-3[each.key].id
}
