# Lab 12 — WebAssembly Containers with Spin

## Environment

Test machine:

    OS: Linux 7.0.0-38-generic x86_64 GNU/Linux
    CPU: 11th Gen Intel(R) Core(TM) i5-11400H @ 2.70GHz
    Logical CPUs: 12

Tool versions:

    Spin 3.6.3
    TinyGo 0.41.0
    Go 1.26.0
    hyperfine 1.19.0
    wasmtime 49.0.3

---

# Task 1 — Spin WebAssembly HTTP Endpoint

The application is located in:

    wasm/moscow-time/

It was scaffolded using the current Spin Go HTTP template:

    spin new -t http-go moscow-time --accept-defaults

This produced the current Spin SDK project structure instead of using the deprecated WAGI configuration.

## main.go

    package main

    import (
        "fmt"
        "net/http"
        "time"

        spinhttp "github.com/spinframework/spin-go-sdk/v2/http"
    )

    func init() {
        spinhttp.Handle(func(w http.ResponseWriter, r *http.Request) {
            if r.Method != http.MethodGet {
                http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
                return
            }

            moscow := time.FixedZone("MSK", 3*60*60)
            now := time.Now().In(moscow)

            w.Header().Set("Content-Type", "application/json")

            fmt.Fprintf(
                w,
                "{\"unix\":%d,\"iso\":%q,\"hour_minute\":%q}\n",
                now.Unix(),
                now.Format(time.RFC3339),
                now.Format("15:04"),
            )
        })
    }

    func main() {}

## spin.toml

    #:schema https://schemas.spinframework.dev/spin/manifest-v2/latest.json

    spin_manifest_version = 2

    [application]
    name = "moscow-time"
    version = "0.1.0"
    authors = ["Abdullojon <ab.muminov@innopolis.university>"]
    description = ""

    [[trigger.http]]
    route = "/time"
    component = "moscow-time"

    [component.moscow-time]
    source = "main.wasm"
    allowed_outbound_hosts = []

    [component.moscow-time.build]
    command = "tinygo build -target=wasip1 -buildmode=c-shared -no-debug -o main.wasm ."
    watch = ["**/*.go", "go.mod"]

## Build

Command:

    spin build

Output:

    Building component moscow-time with `tinygo build -target=wasip1 -buildmode=c-shared -no-debug -o main.wasm .`
    Finished building all Spin components

Resulting artifact:

    main.wasm = 371114 bytes

## Runtime proof

Spin was started using:

    spin up --listen 127.0.0.1:3001

Port 3001 was used because another local application was already bound to port 3000.

Spin reported:

    Serving http://127.0.0.1:3001
    Available Routes:
      moscow-time: http://127.0.0.1:3001/time

Request:

    curl -i http://127.0.0.1:3001/time

Response:

    HTTP/1.1 200 OK
    content-type: application/json

    {"unix":1792086362,"iso":"2026-10-15T20:46:02+03:00","hour_minute":"20:46"}

The timestamp contains the required Moscow UTC+3 offset.

## Design question a — Browser WASM vs server WASM

Browser-targeted WebAssembly expects a browser and JavaScript host environment. A normal Go browser WASM program depends on browser-facing runtime support and JavaScript glue.

The TinyGo `wasip1` target instead produces a module intended for a WASI runtime outside the browser.

The server-side module does not get browser APIs such as the DOM or normal JavaScript APIs. Access to operating-system functionality is instead provided through WASI capabilities supplied by the host.

The benefit is a portable server artifact that can execute under WASI runtimes without requiring a browser and with a more restricted capability model.

## Design question b — Why -buildmode=c-shared?

The Spin HTTP host expects the component to expose the functions required by the Spin Go SDK / wasi-http integration.

Using:

    -buildmode=c-shared

causes TinyGo to export the required handler interface instead of producing an ordinary standalone WASI command.

Without the expected exports, Spin cannot invoke the HTTP handler correctly.

## Design question c — allowed_outbound_hosts = []

This setting gives the component no outbound network capability.

WASI follows a capability-based security model: a component receives only the host resources that are explicitly granted to it. The absence of a capability means the component cannot simply make the corresponding system call directly.

Docker's:

    --network none

also removes normal external network connectivity, but the application is still a Linux process running against the host kernel through the container isolation layer.

WASM reduces the exposed interface further because arbitrary Linux syscalls are not directly available to the module. Network, filesystem and other host interactions must be provided by the WASI host.

## Design question d — TinyGo stdlib limitations

The relevant limitation in this lab was timezone support.

Using:

    time.LoadLocation("Europe/Moscow")

can depend on timezone database data that is not available in a small WASI/TinyGo environment.

Therefore the endpoint uses Moscow's fixed UTC+3 offset:

    time.FixedZone("MSK", 3*60*60)

I also avoided reflection-heavy JSON generation with `map[string]any` and instead formatted the small JSON response explicitly with `fmt.Fprintf`, which is more predictable under TinyGo.

---

# Task 2 — Performance Comparison

## Warm latency methodology

Both services were already running.

Spin:

    http://127.0.0.1:3001/time

Docker:

    http://127.0.0.1:18080/health

Each was benchmarked using:

    hyperfine --warmup 5 --runs 50

Spin result:

    Time (mean ± σ): 5.9 ms ± 0.2 ms
    Range: 5.5 ms ... 6.3 ms

Docker result:

    Time (mean ± σ): 4.4 ms ± 0.2 ms
    Range: 4.0 ms ... 4.9 ms

The exact 50 samples exported by hyperfine were sorted to calculate p50 and p95.

Results:

    Spin:
      p50 = 5.829 ms
      p95 = 6.178 ms

    Docker:
      p50 = 4.428 ms
      p95 = 4.631 ms

Because these commands complete in only a few milliseconds, hyperfine warned that shell startup overhead may influence the absolute numbers. The same methodology was used consistently for both platforms.

## Cold-start methodology

For each sample the runtime was fully stopped.

Timing began immediately before launching the runtime and ended after the first successful HTTP response.

Five samples were collected for each platform.

Spin cold-start samples:

    74.568 ms
    65.413 ms
    80.904 ms
    38.705 ms
    38.849 ms

Spin cold p50:

    65.413 ms

Docker cold-start samples:

    199.120 ms
    216.315 ms
    198.449 ms
    189.925 ms
    197.242 ms

Docker cold p50:

    198.449 ms

## Artifact sizes

Spin WebAssembly component:

    371114 bytes

Lab 6 Docker image:

    5838720 bytes

The Spin artifact is roughly 15.7 times smaller than the Docker image.

## Performance table

| Dimension | Lab 6 Docker | Lab 12 Spin/WASM |
| --- | ---: | ---: |
| Artifact size | 5,838,720 B | 371,114 B |
| Cold start p50 | 198.449 ms | 65.413 ms |
| Warm latency p50 | 4.428 ms | 5.829 ms |
| Warm latency p95 | 4.631 ms | 6.178 ms |

On this machine, Docker was slightly faster for already-warm HTTP requests.

Spin was approximately three times faster in the measured cold-start experiment and produced a much smaller artifact.

## Design question e — What dominates cold start?

For the Docker container, startup includes Docker daemon orchestration, creation of the container's namespaces/cgroups and filesystem view, starting the Linux process, and waiting until the HTTP server is ready.

The image was already local, so the measurement did not include downloading it from a registry.

For Spin, startup includes launching Spin, initializing the Wasmtime-based runtime, loading and instantiating the WASM component, configuring the HTTP trigger, and making the route available.

The WASM runtime has substantially less operating-system process/container setup to perform, which is consistent with the lower cold-start times measured here.

## Design question f — Where WASM wins and where Docker wins

WASM is attractive for:

- small stateless request handlers
- edge functions
- plugin systems
- high-fan-out workloads
- multi-tenant execution
- workloads where startup latency matters
- portable functions that should run across CPU architectures

Docker remains a better fit for:

- long-running stateful services
- applications requiring a large existing Linux ecosystem
- software that depends on arbitrary system calls
- heavy native dependencies
- applications requiring traditional database/network libraries that are not mature under WASI
- teams whose infrastructure is already built around container tooling

The results here demonstrate the trade-off: Spin had a much smaller artifact and better cold-start latency, while the already-running Docker service was slightly faster in the warm HTTP benchmark.

## Design question g — Multi-tenant safety

A WASM module cannot directly invoke arbitrary host syscalls. It receives only capabilities exposed by the runtime.

This makes attacks that depend on reaching unexpected host resources harder.

For example, a compromised HTTP handler without filesystem or outbound-network capabilities cannot simply open arbitrary host files or create arbitrary network sockets.

A container also isolates applications with namespaces, capabilities and other Linux mechanisms, but the contained process still interacts with the shared host kernel syscall interface. A kernel/container escape vulnerability therefore exposes a larger underlying attack surface than a minimal WASI capability interface.

---

# Bonus — Standalone WASI CLI under Wasmtime

A second implementation is stored in:

    wasm-cli/main.go

Unlike the Spin version, it does not use the Spin SDK.

It behaves like a small CGI-style command: request metadata arrives through environment variables and the response is printed to stdout.

## Source

    package main

    import (
        "fmt"
        "os"
        "time"
    )

    func main() {
        method := os.Getenv("REQUEST_METHOD")
        path := os.Getenv("PATH_INFO")

        if method != "GET" || path != "/time" {
            fmt.Println(`{"error":"not found"}`)
            return
        }

        moscow := time.FixedZone("MSK", 3*60*60)
        now := time.Now().In(moscow)

        fmt.Printf(
            "{\"unix\":%d,\"iso\":%q,\"hour_minute\":%q}\n",
            now.Unix(),
            now.Format(time.RFC3339),
            now.Format("15:04"),
        )
    }

## Build

Command:

    tinygo build -o main.wasm -target=wasi -no-debug ./main.go

Artifact size:

    195907 bytes

The standalone CLI module is smaller than the Spin wasi-http component:

    Spin component: 371114 bytes
    WASI CLI:      195907 bytes

## Run

Command:

    wasmtime run \
      --env REQUEST_METHOD=GET \
      --env PATH_INFO=/time \
      main.wasm

Output:

    {"unix":1792086784,"iso":"2026-10-15T20:53:04+03:00","hour_minute":"20:53"}

## Per-invocation benchmark

Command:

    hyperfine \
      --warmup 3 \
      --runs 50 \
      'wasmtime run --env REQUEST_METHOD=GET --env PATH_INFO=/time main.wasm > /dev/null'

Result:

    Time (mean ± σ): 7.3 ms ± 0.5 ms
    Range: 6.7 ms ... 8.7 ms

Calculated percentiles:

    p50 = 7.186 ms
    p95 = 8.269 ms

Each benchmark command launches a new `wasmtime run` process, so this measures the standalone per-invocation execution model.

This should not be interpreted as a direct HTTP latency comparison with Spin because the CLI version does not start an HTTP server or process an HTTP connection.

## Execution-model comparison

| Dimension | Spin wasi-http | Standalone wasmtime CLI |
| --- | ---: | ---: |
| Module size | 371,114 B | 195,907 B |
| Execution model | Persistent HTTP host | New CLI invocation |
| Interface | wasi-http | WASI command / stdout |
| Measured p50 | 5.829 ms warm HTTP | 7.186 ms per invocation |
| HTTP routing | Provided by Spin | None |

Spin additionally had a full-runtime cold-start p50 of:

    65.413 ms

This includes launching Spin and bringing up its HTTP server, so it is not directly equivalent to the standalone Wasmtime invocation measurement.

## Design question h — Why can't the Spin component use bare wasmtime run?

The Spin component is a wasi-http component.

It exports the HTTP handler interface expected by a wasi-http host rather than behaving as a conventional WASI command with a `_start` entrypoint.

Bare:

    wasmtime run

is intended to execute command-style WASI programs.

Therefore the Spin component needs a wasi-http host such as Spin or an appropriate Wasmtime HTTP host, while the standalone bonus module can be executed directly with `wasmtime run`.

## Design question i — What does Spin add over Wasmtime?

Spin uses Wasmtime as its WebAssembly execution engine but adds the application platform around it.

Spin provides:

- an HTTP server
- request routing
- `spin.toml` application manifests
- wasi-http integration
- component lifecycle management
- instance creation/pooling
- component configuration
- outbound-host capability policy
- application-level developer tooling

Bare Wasmtime provides the lower-level WebAssembly runtime.

## Design question j — When does each model fit?

The standalone `wasmtime run` model is suitable for command-oriented or job-style workloads where one invocation performs one operation and exits.

An example would be a sandboxed file transformation or plugin executed once by a parent process.

The Spin wasi-http model is better for HTTP workloads that need routing and repeated requests.

An example would be an edge HTTP API endpoint such as the Moscow-time service in this lab.

Spin avoids requiring the developer to manually construct the HTTP host/routing layer around every WebAssembly component.

---

# Final Result

Task 1:

- Spin 3.x template used
- Go handler built with the current Spin Go SDK
- TinyGo `wasip1` target used
- `-buildmode=c-shared` enabled
- `allowed_outbound_hosts = []`
- `/time` returns valid Moscow-time JSON
- WebAssembly artifact size recorded
- design questions a-d answered

Task 2:

- real warm benchmarks collected
- p50 and p95 calculated
- five Docker cold-start samples collected
- five Spin cold-start samples collected
- artifact sizes compared
- hardware and OS documented
- design questions e-g answered

Bonus:

- standalone WASI CLI implementation created
- builds with TinyGo
- executes successfully under bare Wasmtime
- module sizes compared
- per-invocation latency measured
- Spin and bare Wasmtime execution models compared
- design questions h-j answered
