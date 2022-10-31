# terraform-resource-markdown-table-action

> GitHub Action for generating a markdown table for selected Terraform resources/attributes

## Requirements

The action will automatically run `terraform init` to download provider plugins in order to obtain resource schemas. You must provide any credentials necessary to run `terraform init`. Since this action runs in a container, providing any required credentials as environment variables is recommended.

## Usage

For ease of maintenance, the examples below do not include a version number. In production usage, a version tag should always be specified to avoid unexpected breaking changes.

To generate a markdown table into `README.md` for a module in the repository root:

```yaml
- uses: observeinc/terraform-resource-markdown-table-action
  with:
    resources: |
      - name: my_resource
        attributes:
          - attr_1
          - attr_2
```

To generate a markdown table for a module in a specified directory:

```yaml
- uses: observeinc/terraform-resource-markdown-table-action
  with:
    working_directory: ./path/to/my/module
    resources: ...
```

To avoid writing to disk and handle the resulting markdown directly as an action output:

```yaml
- uses: observeinc/terraform-resource-markdown-table-action
  id: tf-table
  with:
    output-file: ''
    resources: ...
- run: echo $TABLE
  env:
    TABLE: ${{ steps.tf-table.outputs.markdown }}
```

## Limitations

* Data sources are not supported
* Attributes must be defined at the top level and not within blocks
* Attributes must be a primitive type (string, boolean, number)
* Non-static expressions will be printed as _unknown_
