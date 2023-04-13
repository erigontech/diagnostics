# diagnostics system web application

# How to build and run
To build, perform `git clone`, change to the directory with the source code and run:

```
go build .
```

This will create `diagnostics` executable in the same directory.

To run with premade self-signed certificates for TLS (mandatory for HTTP/2), use this command:

```
./diagnostics --tls.cert demo-tls/diagnostics.crt --tls.key demo-tls/diagnostics-key.pem --tls.cacerts demo-tls/CA-cert.pem
```

# How to access from the browser

In the browser, go to the URL `https://localhost:8080/ui`. You browser will likely ask to to accept the risks (due to self-signed certificate), do that.

# How to run an Erigon node that can be connected to the diagnosgics system

For an Erigon node to be connected to the diagnostics system, it needs to expose metrics using this command line flag:

```
--metrics
```

By default the metrics are exposed on `localhost` and port `6060`. In order to expose them on a different networking interface and/or different port,
the following command line flags can be used:

```
--metrics.addr <IP of interface> --metrics.port <port>
```

In order to check is the metrics are exposed on given interface or port, you can check it in the browser by going to
```
http://<metrics.addr>:<metrics.port>/debug/metrics/prometheus
```

If metrics are exposed, textual representation of metrics will be displayed in the browser.

# How to connect Erigon node to the diagnostics system

First, in the brower window, create a new operator session. Choose an arbitraty name. In real operations, one would choose the name
that can be easily correlate to the node being supported, for example, name or pseudonym of the person or the company operating the node.

![create new operation session](/images/create_new_session.png)

After new session is created, it will be allocated a unique 8-digit PIN number. The pin is then displayed together with the session number on the screen.
Currently generation of PIN numbers is not secure and always follows the same sequence, which makes testing easier. For example, the first
allocated session PIN is always `47779410`.

Next, in a console window, run the following command, specfying the session PIN at the end of the `--diagnostics.url` command line flag.
Since the web site is using self-signed certificate without properly allocated CName, one needs to use `--insecure` flag to be able to connect.

```
./build/bin/erigon support --metrics.urls http://metrics.addr:metrics.port/debug/metrics --diagnostics.url https://localhost:8080/support/47779410 --insecure
```
