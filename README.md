# Calico Terraform Provider

## About
- For use with Calico 2.x with the etcd backend

## Install
Due to the large amount of dependencies from libcalico-go and it's usage of glide for dep management, the install is a bit more than just a go get.

```
mkdir -p $GOPATH/src/github.com/bolcom
git clone https://github.com/bolcom/terraform-provider-calico.git $GOPATH/src/github.com/bolcom/terraform-provider-calico
cd $GOPATH/src/github.com/bolcom/terraform-provider-calico
./build_for_terraform_version.sh 0.7.11 #insert your terraform version here
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
Optional:
- backend_etcd_scheme: default: http
- backend_etcd_endpoints: multiple etcd endpoints separated by comma
- backend_etcd_username
- backend_etcd_password
- backend_etcd_keyfile: File location keyfile
- backend_etcd_certfile: File location certfile
- backend_etcd_cacertfile: File location cacert

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
### Policy
```
resource "calico_policy" "mypolicy" {
  name = "mypolicy"
  spec {
    order = 100
    selector = "globalpolicy == 'test123'"
    ingress {
      rule {
        action = "deny"
        protocol = "tcp"
        source {
          net = "10.0.0.0/24"
          selector = "mykey == 'test'"
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
### IP Pools
```
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
```
### BGP Peers
```
resource "calico_bgppeer" "mybgppeer" {
  scope = "node"
  node = "rack1-host1"
  peerIP = "192.168.1.1"
  spec {
    asNumber = "63400"
  }
}
```
### Nodes
```
resource "calico_node" "mynode" {
  name = "node-hostname"
  spec {
    bgp {
      asNumber = "64512"
      ipv4Address = "10.244.0.1"
      ipv6Address = "2001:db8:85a3::8a2e:370:7334"
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
