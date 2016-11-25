resource "calico_profile" "myprofile" {
  name = "myprofile"
  labels = { endpointlabel = "myvalue" }
  spec {
    ingress {
      rule {
        action = "deny"
        protocol = "tcp"
        source {
          net = "10.0.0.0/24"
          selector = "profile == 'test'"
          ports = ["1:10", "20:30"]
          notPorts = ["40:60"]
        }
        icmp {
          code = 100
          type = 101
        }
      }
      rule {
        action = "allow"
        protocol = "udp"
        source {
          net = "11.0.0.0/24"
        }
      }
    }
    egress {
      rule {
        action = "deny"
        protocol = "tcp"
        source {
          net = "12.0.0.0/24"
        }
      }
      rule {
        action = "allow"
        protocol = "udp"
        source {
          net = "13.0.0.0/24"
        }
      }
    }
  }
}
