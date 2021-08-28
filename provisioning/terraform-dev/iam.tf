data "aws_iam_policy" "IsuAdmin" {
  name = "IsuAdmin"
}

resource "aws_iam_user" "ghaction" {
  name                 = "ghaction-final"
  permissions_boundary = data.aws_iam_policy.IsuAdmin.arn
}

resource "aws_iam_user_policy" "ghaction-s3" {
  user   = aws_iam_user.ghaction.name
  name   = "s3"
  policy = data.aws_iam_policy_document.ghaction-s3.json
}

data "aws_s3_bucket" "artifacts" {
  bucket = "isucon11-artifacts"
}

data "aws_iam_policy_document" "ghaction-s3" {
  statement {
    effect = "Allow"
    actions = [
      "s3:GetObject",
    ]
    resources = [
      "${data.aws_s3_bucket.artifacts.arn}/supervisor/*",
    ]
  }
}

resource "aws_iam_user_policy" "ghaction-packer" {
  user   = aws_iam_user.ghaction.name
  name   = "ghaction-packer"
  policy = data.aws_iam_policy_document.ghaction-packer.json
}

data "aws_iam_policy_document" "ghaction-packer" {
  statement {
    effect = "Allow"
    actions = [
      "ec2:AttachVolume",
      "ec2:AuthorizeSecurityGroupIngress",
      "ec2:CopyImage",
      "ec2:CreateImage",
      "ec2:CreateKeypair",
      "ec2:CreateSecurityGroup",
      "ec2:CreateSnapshot",
      "ec2:CreateTags",
      "ec2:CreateVolume",
      "ec2:DeleteKeyPair",
      "ec2:DeleteSecurityGroup",
      "ec2:DeleteSnapshot",
      "ec2:DeleteVolume",
      "ec2:DeregisterImage",
      "ec2:DescribeImageAttribute",
      "ec2:DescribeImages",
      "ec2:DescribeInstances",
      "ec2:DescribeInstanceStatus",
      "ec2:DescribeRegions",
      "ec2:DescribeSecurityGroups",
      "ec2:DescribeSnapshots",
      "ec2:DescribeSubnets",
      "ec2:DescribeTags",
      "ec2:DescribeVolumes",
      "ec2:DetachVolume",
      "ec2:GetPasswordData",
      "ec2:ModifyImageAttribute",
      "ec2:ModifyInstanceAttribute",
      "ec2:ModifySnapshotAttribute",
      "ec2:RegisterImage",
      "ec2:RunInstances",
      "ec2:StopInstances",
      "ec2:TerminateInstances",
      // Spot
      "ec2:CreateLaunchTemplate",
      "ec2:DeleteLaunchTemplate",
      "ec2:CreateFleet",
      "ec2:DescribeSpotPriceHistory",
    ]
    resources = ["*"]
  }
}
