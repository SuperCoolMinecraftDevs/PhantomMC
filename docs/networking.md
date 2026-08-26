# Networking

In stream and pinned modes the machine cannot do anything useful until it has an
internet connection, so network bring up is on the critical path and has to
handle setups considerably worse than a plain DHCP lease.

## Why not PXE or iPXE

The original design used iPXE to fetch the payload over the network. It was
abandoned before any code was written. iPXE runs before Linux does, which means
it has its own network stack, its own drivers and its own TLS. In practice that
means no modern wifi, frequent trouble with anything beyond a straightforward
DHCP server, and TFTP as the transport, which relies on broadcast behaviour that
consumer routers and hand rolled NAT boxes handle badly.

The alternative is to let Linux do it. A small kernel is on the stick, it boots
in seconds, and from that point the full Linux network stack is available:
proper drivers, `iwd` for wifi, and ordinary HTTPS for the fetch. Every awkward
network becomes an ordinary Linux networking problem, which is a solved category.

See [alternatives.md](alternatives.md) for the other approaches considered.

## Bring up order

```
  ┌─ ethernet present and carrier up? ────────── yes ──▶ DHCP ──▶ verify
  │                                                                 │
  no                                                                │
  │                                                                 │
  ├─ wifi hardware present? ─── no ──▶ error: no usable interface    │
  │                                                                 │
  yes                                                               │
  │                                                                 │
  ▼                                                                 │
  scan, present a list, take a passphrase, associate ──▶ DHCP ──────┤
                                                                    │
                                       ┌────────────────────────────┘
                                       ▼
                              reachability check
                                       │
                        ┌──────────────┴──────────────┐
                        │                             │
                   reachable                    captive portal
                        │                             │
                        ▼                             ▼
                     proceed                  show the portal notice
```

Ethernet is tried first and without prompting, because the overwhelmingly common
case is a desktop with a cable in it and asking would be noise.

## Wifi

Handled by `iwd` rather than `wpa_supplicant` plus NetworkManager. It is
substantially smaller, it has a sane control interface, and it does not drag in
a dependency chain built for a desktop session.

The passphrase is entered on each boot. There is nowhere to store it, and
storing it would break the promise the project is built on.

WPA2 and WPA3 personal are the target. WPA Enterprise, which needs certificates
and an identity, is not supported yet and is tracked separately.

## Awkward setups

The design assumption is that the network between the machine and the internet
may be strange. Custom NAT boxes, a Raspberry Pi doing routing, unusual DNS,
double NAT, VLANs on the wire.

The mitigation is to depend on as little as possible. The agent needs outbound
HTTPS on port 443 and working DNS. It does not need inbound connections, does
not use TFTP or any broadcast protocol, does not care about the address range it
is handed, and does not assume the gateway is also the resolver.

Where DNS is broken but connectivity works, `systemd-resolved` is configured
with fallback resolvers so a misconfigured local resolver does not take the boot
down with it.

## Captive portals

Common in hotels, dormitories and conference venues. Detection is a request to a
known endpoint with a known response, and a mismatch means something is
intercepting traffic.

There is no browser to complete a portal in. The intended behaviour is to detect
the condition and say so plainly, suggesting the user complete the portal on
another device on the same network first. That is the honest answer, and it is
far better than a silent thirty second timeout on a mod download.

## Offline operation

Prefetch mode carries everything on the stick and needs no network at all,
provided the manifest also uses offline authentication. A prefetch stick with
`auth.mode: microsoft` still needs to reach Microsoft. `Manifest.RequiresNetwork`
computes this, so the agent can tell the user before the boot rather than during
it.
