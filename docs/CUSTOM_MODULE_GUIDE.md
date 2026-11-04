# Writing Custom Module Adapters for `gcp-tf-forge`

`gcp-tf-forge` allows organizations to import their existing Google Cloud infrastructure directly into their **own internal custom Terraform modules** instead of standard raw HCL resources.

---

## 1. Quick Example: Custom VPC Module

Suppose your platform team maintains a custom VPC module at `./modules/company-vpc`:

```
modules/company-vpc/
├── main.tf
├── variables.tf
└── outputs.tf
```

### The Module Definition (`variables.tf`):
```hcl
variable "vpc_name" {
  type        = string
  description = "The name of the VPC network"
}

variable "subnetwork_list" {
  type = list(object({
    name = string
    cidr = string
  }))
}
```

---

## 2. Running with Automatic AST Inspection

You can simply point `gcp-tf-forge` to your modules folder:
```bash
gcp-tf-forge export \
  --project=my-prod-project \
  --custom-modules=./modules/
```
The AST analyzer will:
1. Inspect `variables.tf` and match `vpc_name` to raw `name`.
2. Inspect the resource address inside `main.tf` to construct the exact Terraform 1.5+ `import` block:
   ```hcl
   import {
     to = module.company_vpc.google_compute_network.this
     id = "projects/my-prod-project/global/networks/my-vpc"
   }
   ```

---

## 3. Explicit Declarative Mapping Override (`mapping.yaml`)

If your module has unusual variable naming or complex local transformations, create a lightweight `module-spec.yaml`:

```yaml
modules:
  - name: "company_vpc"
    source: "git::https://github.com/my-org/tf-modules.git//vpc"
    version: "~> 2.0"
    primary_resource: "google_compute_network"
    internal_resource_name: "this"
    field_mappings:
      name: "vpc_name"
    bundled_children:
      - resource_type: "google_compute_subnetwork"
        internal_resource_name: "subnets"
        map_to_input: "subnetwork_list"
        index_key_pattern: "${region}/${name}"
        field_mappings:
          name: "name"
          ip_cidr_range: "cidr"
```

Pass the spec:
```bash
gcp-tf-forge export \
  --project=my-prod-project \
  --custom-modules=./module-spec.yaml
```
