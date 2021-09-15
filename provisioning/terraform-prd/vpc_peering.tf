data "aws_vpc" "portal" {
  id = "vpc-04f36e0596c6daf7f"
}

data "aws_route_table" "portal-public" {
  vpc_id = data.aws_vpc.portal.id
  filter {
    name   = "tag:Name"
    values = ["isucon11-public"]
  }
}

data "aws_route_table" "portal-private" {
  vpc_id = data.aws_vpc.portal.id
  filter {
    name   = "tag:Name"
    values = ["isucon11-private"]
  }
}

resource "aws_vpc_peering_connection" "portal" {
  vpc_id      = aws_vpc.main.id
  peer_vpc_id = data.aws_vpc.portal.id
  auto_accept = true
}

resource "aws_route" "bench-to-portal" {
  route_table_id            = aws_route_table.bench.id
  destination_cidr_block    = data.aws_vpc.portal.cidr_block
  vpc_peering_connection_id = aws_vpc_peering_connection.portal.id
}

resource "aws_route" "portal-public-to-main" {
  route_table_id            = data.aws_route_table.portal-public.id
  destination_cidr_block    = aws_vpc.main.cidr_block
  vpc_peering_connection_id = aws_vpc_peering_connection.portal.id
}

resource "aws_route" "portal-private-to-main" {
  route_table_id            = data.aws_route_table.portal-private.id
  destination_cidr_block    = aws_vpc.main.cidr_block
  vpc_peering_connection_id = aws_vpc_peering_connection.portal.id
}
