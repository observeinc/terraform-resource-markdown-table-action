resource "observe_monitor" "foo" {
  name = "Foo"
  description = "Bar"

  workspace = var.workspace.oid
  
  inputs = {
    "key" = "value"
  }
}
