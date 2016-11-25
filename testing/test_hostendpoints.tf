resource "calico_hostendpoint" "myendpoint" {
  name = "myendpoint"
  node = "my-endpoint-001"
  interface = "eth0"
  expected_ips = ["127.0.0.1"]
  profiles = ["endpointprofile"]
  labels = { endpointlabel = "myvalue" }
}
