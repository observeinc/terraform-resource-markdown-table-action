resource "observe_monitor" "foo" {
  name = "Foo"
  description = "Bar"
  workspace = "1"
  inputs = {
    "key" = "value"
  }
}
