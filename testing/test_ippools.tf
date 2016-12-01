resource "calico_ippool" "myippool" {
  cidr = "10.1.0.0/16"
  spec {
    ipip {
      enabled = "true"
    }
    nat-outgoing = "true"
    disabled = "true"
  }
}
