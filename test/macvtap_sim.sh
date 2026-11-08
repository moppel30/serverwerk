#!/bin/sh
# Baut in WSL eine "macvtap-VM" nach: Netz-Namespace mit macvlan-Karte am selben Anschluss.
set -e
P=eth0
GIP=${1:-172.21.190.77}
ip netns del guest 2>/dev/null || true
ip link del mvt0 2>/dev/null || true
ip link del sw-shim 2>/dev/null || true
ip route del $GIP/32 2>/dev/null || true
ip link add mvt0 link $P type macvtap mode bridge     # das sieht der Host wie eine libvirt-VM
ip netns add guest
ip link add g0 link $P type macvlan mode bridge
ip link set g0 netns guest
PFX=$(ip -o -4 addr show $P | awk '{print $4}' | cut -d/ -f2)
ip netns exec guest ip addr add $GIP/$PFX dev g0
ip netns exec guest ip link set lo up
ip netns exec guest ip link set g0 up
echo "Gast $GIP/$PFX bereit"
