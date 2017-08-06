# zone-nameservers

Walk the DNS tree to find which name servers a particular zone uses. Mimics "dig +trace", but written in Go as an experiment.

# Building

After a git clone;

```
$ go build
$ ./zone-nameservers YOURDOMAIN.TLD
```

# Examples

Here's what it looks like for [dnsspy.io](https://dnsspy.io).

```
./zone-nameservers dnsspy.io
Retrieving list of root nameservers:
 - a.root-servers.net.
 - b.root-servers.net.
 - c.root-servers.net.
 - d.root-servers.net.
 - e.root-servers.net.
 - f.root-servers.net.
 - g.root-servers.net.
 - h.root-servers.net.
 -> i.root-servers.net.
 - j.root-servers.net.
 - k.root-servers.net.
 - l.root-servers.net.
 - m.root-servers.net.


Finding nameservers for zone 'io.' using parent nameserver 'i.root-servers.net.'
 - a0.nic.io.
 - b0.nic.io.
 - c0.nic.io.
 - ns-a1.io.
 - ns-a2.io.
 -> ns-a3.io.
 - ns-a4.io.


Finding nameservers for zone 'dnsspy.io.' using parent nameserver 'ns-a3.io.'
 - ns1.nucleus.be.
 - ns2.nucleus.be.
 - ns3.nucleus.be.
 - ns4.nucleus.be.
 ```

The arrow represents which nameserver from the parent was used to query for details of the child zone.

# Why?

Why not just query directly for `NS` records, you ask? Not everyone keeps those up-to-date and they often return outdated or wrong information, as nameservers change without modifying the `NS` records to reflect that.

In other words: the only _absolutely_ way to find our which nameservers a particular zone uses, you have to walk the DNS tree.

# Credits

This code is initially based on the [check-soa](https://github.com/miekg/exdns/tree/master/check-soa) script by [miekg](https://github.com/miekg).
