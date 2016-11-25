# Calico Terraform Provider

## About
- For use with Calico 2.x with the etcd backend
- Only Hostendpoints and Profiles supported, more coming soon

## Install
Due to the large amount of dependencies from libcalico-go and it's usage of glide for dep management, the install is a bit more than just a go get.
```
$ go get github.com/bolcom/terraform-provider-calico
$ glide install # github.com/Masterminds/glide
$ go build
```

## Usage

### Provider Configuration
provider.tf
```
provider "calico" {
  backend_type = "etcdv2"
  backend_etcd_authority = "192.168.56.20:2379"
}
```
### Host Endpoint
```
resource "calico_hostendpoint" "myendpoint" {
  name = "myendpoint"
  node = "my-endpoint-001"
  interface = "eth0"
  expected_ips = ["127.0.0.1"]
  profiles = ["endpointprofile"]
  labels = { endpointlabel = "myvalue" }
}
```
### Profile
```
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
```

## Testing
The script test.sh will:
- download calicoctl and terraform
- build terraform-provider-calico
- spin up a container with etcd (docker-compose)
- pull tests out of testing/test_*
- do a terraform apply of the TF file
- use calicoctl to get the result
- compare it with the prestored results in the test_*.yaml file
