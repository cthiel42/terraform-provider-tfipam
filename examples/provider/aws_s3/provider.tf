terraform {
  required_providers {
    tfipam = {
      source  = "cthiel42/tfipam"
      version = "1.2.0"
    }
  }
}

# Example 1: Using explicit AWS credentials
provider "tfipam" {
  storage_type         = "aws_s3"
  s3_region            = "us-east-1"
  s3_bucket_name       = "my-tfipam-bucket"
  s3_object_key        = "ipam-storage.json" # Optional: defaults to "ipam-storage.json"
  s3_access_key_id     = "AKIAABCDEFGHEXAMPLE"
  s3_secret_access_key = "ACCESSKEYEXAMPLE1234567890"
  s3_endpoint_url      = "https://s3.example.com" # Optional: for S3 compatible services like MinIO or LocalStack
  # s3_session_token    = "token"                 # Optional: for temporary credentials
}

# Example 2: Using default AWS credential chain (IAM role, env vars, ~/.aws/credentials)
# provider "tfipam" {
#   storage_type   = "aws_s3"
#   s3_region      = "us-east-1"
#   s3_bucket_name = "my-tfipam-bucket"
#   s3_object_key  = "ipam-storage.json"
#   # Credentials will be loaded from:
#   # 1. Environment variables (AWS_ACCESS_KEY_ID, AWS_SECRET_ACCESS_KEY)
#   # 2. Shared credentials file (~/.aws/credentials)
#   # 3. IAM role (if running on EC2, ECS, Lambda, etc.)
# }

resource "tfipam_pool" "example" {
  name = "pool_example"
  cidrs = [
    "10.0.0.0/24",
    "10.5.0.0/24"
  ]
}