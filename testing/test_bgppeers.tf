resource "calico_bgppeer" "mybgppeer" {
  scope = "node"
  node = "rack1-host1"
  peerIP = "192.168.1.1"
  spec {
    asNumber = "63400"
  }
}
