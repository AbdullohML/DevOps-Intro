# Lab 4 Submission — OS & Networking

## Task 1 — Trace a Request End-to-End

### 1.1 QuickNotes and packet capture

QuickNotes was started locally:

```text
$ cd app
$ go run .
2026/09/17 23:09:08 quicknotes listening on :8080 (notes loaded: 5)
```

Packet capture:

```bash
sudo tcpdump -i lo -nn -s 0 -A 'tcp port 8080' -w lab4-trace.pcap &
TCPDUMP_PID=$!
```

POST request:

```bash
curl -v -X POST http://localhost:8080/notes \
  -H 'Content-Type: application/json' \
  -d '{"title":"trace me","body":"in flight"}'
```

Response:

```text
HTTP/1.1 201 Created

{"id":7,"title":"trace me","body":"in flight","created_at":"2026-09-17T20:12:10.70533445Z"}
```

The capture was stopped with:

```bash
sudo kill $TCPDUMP_PID
wait $TCPDUMP_PID 2>/dev/null
```

The resulting capture contained 10 packets.

### 1.2 Packet capture analysis

The capture was decoded with:

```bash
sudo tcpdump -r lab4-trace.pcap -nn -A | tee lab4-trace.txt
```

Relevant packets:

```text
23:12:10.705052 ::1.50594 > ::1.8080 Flags [S]
SYN — client starts the TCP three-way handshake.

23:12:10.705074 ::1.8080 > ::1.50594 Flags [S.]
SYN/ACK — server acknowledges the connection request.

23:12:10.705086 ::1.50594 > ::1.8080 Flags [.]
ACK — TCP handshake completed.

23:12:10.705171 HTTP: POST /notes HTTP/1.1
Host: localhost:8080
Content-Type: application/json
Content-Length: 39

{"title":"trace me","body":"in flight"}

23:12:10.705558 HTTP: HTTP/1.1 201 Created
Content-Type: application/json
Content-Length: 92

{"id":7,"title":"trace me","body":"in flight","created_at":"2026-09-17T20:12:10.70533445Z"}

23:12:10.705728 ::1.50594 > ::1.8080 Flags [F.]
23:12:10.705765 ::1.8080 > ::1.50594 Flags [F.]

FIN packets — TCP connection closed cleanly.
```

The complete decoded capture is included in `lab4-trace.txt`.

### 1.3 Five debugging commands

#### 1. Listening socket

```bash
ss -tlnp | grep :8080
```

```text
LISTEN 0 4096 *:8080 *:* users:(("quicknotes",pid=479414,fd=3))
```

**Decision:** QuickNotes is listening on TCP port 8080.

#### 2. Routes

```bash
ip route show
```

```text
default via 10.91.48.1 dev wlp0s20f3 proto dhcp src 10.91.54.209 metric 600
10.91.48.0/20 dev wlp0s20f3 proto kernel scope link src 10.91.54.209 metric 600
172.17.0.0/16 dev docker0 proto kernel scope link src 172.17.0.1 linkdown
172.18.0.0/16 dev br-e219829b8597 proto kernel scope link src 172.18.0.1
```

**Decision:** The host has a valid default route through Wi-Fi and Docker bridge routes.

#### 3. Reachability

```bash
mtr -rwc 5 localhost
```

```text
Start: 2026-09-17T23:14:22+0300
HOST: lock      Loss%   Snt   Last   Avg  Best  Wrst StDev
  1.|-- localhost  0.0%     5    0.0   0.1   0.0   0.1   0.0
```

**Decision:** localhost is reachable with 0% packet loss.

#### 4. DNS

```bash
dig +short example.com @1.1.1.1
```

```text
8.6.112.0
8.47.69.0
```

**Decision:** DNS resolution through the specified resolver works.

#### 5. Logs

```bash
journalctl --user -u quicknotes -n 20 || true
```

```text
-- No entries --
```

**Decision:** There are no user-systemd entries because QuickNotes was started manually with `go run .`, not as a systemd user service.

### What I would check first if QuickNotes returned 502

A `502 Bad Gateway` usually means that a proxy or gateway is reachable but cannot get a valid response from its upstream service. I would first check whether QuickNotes is running and listening on the expected address and port using `ss -tlnp`, then call `/health` directly with `curl`. If QuickNotes works directly, I would check the reverse-proxy upstream configuration and logs. If it does not, I would continue checking the process, port binding, firewall, routing, and application logs.

---

## Task 2 — Outside-In Debugging on a Broken Deploy

### 2.1 Broken instance

A second QuickNotes process was started while port 8080 was already occupied.

```bash
ADDR=:8080 go run . 2>&1 | tee /tmp/qn-broken.log
```

Failure:

```text
2026/09/17 23:16:31 quicknotes listening on :8080 (notes loaded: 7)
2026/09/17 23:16:31 listen: listen tcp :8080: bind: address already in use
exit status 1
```

**Root cause:** another QuickNotes process already owned TCP port 8080, so the new process could not bind to it.

### 2.2 Outside-in debugging chain

#### Step 1 — Is it running?

```bash
ps -ef | grep quicknotes
```

```text
abdulll+ 486569 486530 0 23:15 pts/2 00:00:00 /home/abdullloh/.cache/go-build/4e/4e05ba94fff0a11e448e4f9bd8aa03665333d5752a423888e44c96515a77d783-d/quicknotes
abdulll+ 488391 479616 0 23:17 pts/2 00:00:00 grep --color=auto quicknotes
```

**Decision:** a QuickNotes process is already running.

#### Step 2 — Is it listening?

```bash
ss -tlnp | grep 8080
```

```text
LISTEN 0 4096 *:8080 *:* users:(("quicknotes",pid=486569,fd=3))
```

**Decision:** QuickNotes already owns port 8080. This explains the bind failure.

#### Step 3 — Is it reachable?

```bash
curl -s -o /dev/null -w "%{http_code}\n" http://localhost:8080/health
```

```text
200
```

**Decision:** the existing QuickNotes instance is healthy and reachable.

#### Step 4 — Is the firewall blocking it?

```bash
sudo iptables -L -n -v 2>/dev/null || sudo nft list ruleset 2>/dev/null || true
```

Relevant output:

```text
Chain INPUT (policy ACCEPT 0 packets, 0 bytes)
Chain OUTPUT (policy ACCEPT 0 packets, 0 bytes)
```

Docker-specific forwarding rules were also present.

**Decision:** the firewall is not blocking localhost access to QuickNotes.

#### Step 5 — Does DNS work?

```bash
dig +short localhost
```

```text
127.0.0.1
```

**Decision:** localhost resolves correctly, so DNS is not the cause.

### 2.3 Repair and re-verification

The conflicting processes were stopped:

```bash
kill 486530 486569 2>/dev/null || true
sleep 1
ss -tlnp | grep 8080 || echo "port 8080 is free"
```

Output:

```text
2026/09/17 23:18:32 shutting down
port 8080 is free
```

QuickNotes was restarted:

```bash
ADDR=:8080 go run . &
sleep 1
curl -s http://localhost:8080/health
```

Output:

```text
2026/09/17 23:18:38 quicknotes listening on :8080 (notes loaded: 7)
{"notes":7,"status":"ok"}
```

**Decision:** the port conflict was removed and QuickNotes became healthy again.

### Mini-postmortem

The failure happened because two processes attempted to bind to the same TCP port. Although the immediate cause was a duplicate process, this type of failure is systemic because service startup depends on correct process and port ownership. It can happen because of duplicate deployments, stale processes, incorrect configuration, or manually started services.

A process manager such as systemd can make service ownership explicit and prevent uncontrolled duplicate instances. Deployment scripts can also check whether the expected port is already occupied before starting a service. Centralized logs, health checks, and monitoring would make the failed startup immediately visible. The aim is to make service lifecycle and port ownership observable instead of depending on manual debugging.
