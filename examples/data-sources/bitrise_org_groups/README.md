# Bitrise Organization Groups Data Source

This example shows how to retrieve all groups from a Bitrise organization.

## Usage

Replace `my-organization` with your actual organization slug.

```bash
terraform init
terraform plan
terraform apply
```

## Output

The data source will output:
- A list of all groups in the organization
- Each group's slug and name
