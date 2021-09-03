resource "aws_vpc" "main" {
  cidr_block = "10.14.0.0/17"

  tags = {
    Name = "final-pre"
  }
}

resource "aws_internet_gateway" "main" {
  vpc_id = aws_vpc.main.id
}

###
# Contestant
###

resource "aws_subnet" "contestant" {
  for_each = toset(var.team_ids)

  availability_zone       = "ap-northeast-1a"
  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(aws_vpc.main.cidr_block, 7, index(var.team_ids, each.key) + 1)
  map_public_ip_on_launch = false

  tags = {
    Name = format("final-pre-contestant-%02d", tonumber(each.key))
  }
}

resource "aws_route_table" "contestant" {
  vpc_id = aws_vpc.main.id
  tags = {
    Name = "final-pre-contestant"
  }
}
resource "aws_route" "contestant-default" {
  route_table_id         = aws_route_table.contestant.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.main.id
}

resource "aws_route_table_association" "contestant" {
  for_each       = toset(var.team_ids)
  subnet_id      = aws_subnet.contestant[each.key].id
  route_table_id = aws_route_table.contestant.id
}

###
# Benchmarker
###

resource "aws_subnet" "bench" {
  availability_zone       = "ap-northeast-1a"
  vpc_id                  = aws_vpc.main.id
  cidr_block              = cidrsubnet(aws_vpc.main.cidr_block, 7, 0)
  map_public_ip_on_launch = true

  tags = {
    Name = "final-pre-bench"
  }
}

resource "aws_route_table" "bench" {
  vpc_id = aws_vpc.main.id
  tags = {
    Name = "final-pre-bench"
  }
}
resource "aws_route" "bench-default" {
  route_table_id         = aws_route_table.bench.id
  destination_cidr_block = "0.0.0.0/0"
  gateway_id             = aws_internet_gateway.main.id
}

resource "aws_route_table_association" "bench" {
  subnet_id      = aws_subnet.bench.id
  route_table_id = aws_route_table.bench.id
}
