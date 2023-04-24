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

First, in the browser window, create a new operator session. Choose an arbitraty name. In real operations, one would choose the name
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

# Architecture of diagnostics system

Following diagram shows schematicaly how the process of diagnostics works. Erigon nodes that can be diagnosed, need to be running with `--metrics` flag.
Diagnostics system (HTTP/2 web site) needs to be running somewhere. For the public use, it can be a web site managed by erigon team, for example. For
personal and testing use, this can be locally run web site with self-signed certificates.

In order to connect Erigon node to the Diagnostics system, user needs to start a process with a command `erigon support`, as described earlier.
The initiations of network connections are shown as solid single arrows. One can see that `erigon support` initiates connections to both Erigon node
and the diagnostics system. Then, it uses the feature of HTTP/2 called "duplex connection", to create a logical tunnel that allows diagnostics system
to make HTTP requests to the Erigon node, and receive the information exposed on the metrics port. The URLs used to connect `erigon support` to the
diagnostics system, start with `/support/` prefix, followed by the PIN of the session. In the code inside `cmd/root.go`, this corresponds to the
`BridgeHandler` type.

Operators (those who are trying to assists the Erigon node users) also access Diagnosics system, but in the form of User Interface, built using HTML
and Javascript. The URLs used for such access, start with `ui/` prefix. In the code inside `cmd/root.go`, this corresponds to the `UiHandler` type.

![diagnostics system architecture](/images/diagnostics.drawio.png)

# Currently implemented diagnostics

## Code version

Operator can look at the code version that Erigon node has been built with. The corresponding code in erigon is in the file `diagnostics/versions.go`.
The code on the side of the diagnostics system is spread across files `cmd/ui_handler.go` (invocation of `processVersions` function), 
`cmd/versions.go`, `assets/template/session.html` (template in the format of `html/template` package, the part where the button `Fetch Versions` is defined with
the javascript handler), `assets/script/session.js` (function `fetchContent`), `assets/template/versions.html` (html template
for the content fetched by the `fetchContent` javascript function and inserted into the HTML div element).

![versions](/images/versions.png)

## Command line arguments

Operator can look at the command line arguments that were used to launch erigon node. The corresponding code in erigon is in the file `diagnostics/cmd_line.go`.
The code on the side of the diagnostics system is spread across files `cmd/ui_handler.go` (invocation of `processCmdLineArgs` function), 
`cmd/cmd_line.go`, `assets/template/session.html` (html template, the part where the button `Fetch Cmd Line` is defined with
the javascript handler), `assets/script/session.js` (function `fetchContent`), `assets/template/cmd_line.html` (html template
for the content fetched by the `fetchContent` javascript function and inserted into the HTML div element).

![cmd line](/images/cmd_line.png)

## Logs

Since version 2.43.0, erigon nodes write logs by default with `INFO` level into `<datadir>/logs` directory, there is log roation. Using diagnosics system,
these logs can be looked at and downloaded to the operator's computer. Viewing the logs is one of the most frequent requests of the operator to the user,
and it makes sense to make this process much more convinient and efficient. The corresponding code in erigon is in the file `diagnostics/log_access.go`.
Note that the codes does not give access to any other files in the file system, only to the directory dedicated to the logs.
The code on the side of the diagnostics system is spread across files `cmd/ui_handler.go` (invocation of `processLogPart` and `transmitLogFile` functions), 
`cmd/logs.go`, `assets/template/session.html` (html template, the part where the button `Fetch Logs` is defined with
the javascript handler), `assets/script/session.js` (function `fetchContent`), `assets/template/log_list.html` (html template
for the content fetched by the `fetchContent` javascript function and inserted into the HTML div element), `assets/template/log_read.html` (html template
for the buttons `Head`, `Tail` and `Clear` with the invocations of `fetchLogPart` and `clearLog` javascript functions, as well as construction of the
HTML link that activates the download of a log file). The download of a log file is implemented by the `transmitLogFile` function inside `cmd/logs.go`.

![logs](/images/logs.png)

## Reorg scanner

This is the first very crude example of how diagnostics system can access Erigon node's database remotely, via `erigon support` tunnel. Re-orgs can be identified by
the presence of multiple block headers with the same block height but different block hashes.

One of the ideas for the further development of the diagnostics system is the addition of many more such useful "diagnostics scripts", that could be run against
Erigon's node's database, to check the state of the node, or certain inconsistencies etc.

The corresponding code in erigon is in the file `diagnostics/db_access.go`, and it relies on a feature recently added to the Erigon's code, which is
`mdbx.PathDbMap()`, the global function that retuns the mapping of all currently open MDBX environments (databases), keyed by the paths to their directories in the filesystem.
This allows `db_access.go` to create a read-only transaction for any of these environments (databases) and provide remote reading by the diagnostics system.

The code on the side of the diagnostics system is `cmd/reorgs.go`. The function `findReorgs` generates HTML piece by piece, executing two different html templates
(`assets/template/reorg_spacer.html` and `assets/template/reorg_block.html`). These continously generated HTML lines are picked up by javascript function `findReorgs`
in file `assets/script/session.js`, which appends them to `innerHTML` field of the div element. This creates an effect of animation, notifying the operator of the
progress of the scanning for reorgs (with spacer html pieces, one for each 1000 blocks), and showing intermediate results of the scan (with block html pieces,
one for each reorged block found).

![scan reorgs](/images/scan_reorgs.png)

# Block Body Download

This is the first crude example of monitoring an algorithms involving many items (in that case block bodies) transitioning through the series of states.

![body download](/images/body_download.png)

# Ideas for possible improvements

If you are looking at this because you would like to apply to be a part of Erigon development team, the best you can do is to try to first run the
diagnostics system locally as described above, then study the code in the repository and think of a way to improve it. This repository has been
intitially created by a person with very little experience in web server development, web design, javascript, and, more crucially, it has been
created in a bit of a rush.
Therefore, there should be a lot of things that can be improved in terms of best practices, more pleasant user interface, code simplicity, etc.

There are some functional improvements that could be quite useful, for example:

* Reorg scanner is very basic and it does not have a concept of a "deep" reorg (deeper than 1 block). For such situations, it will just show the consequitive
block numbers as all havign a reorg. It would be better to aggregate these into deep reorgs, and also perhaps show if there are more than 1 branch at each
reorg point.
* For the reorg scanner, add the ability to click on the block numbers and get more information about that particular reorg, for example, block producers
for each of the block participating in the reorg, or difference in terms of transactions.
* Any sessions created via User Interface, stay in the server forever and are never cleaned up, so theoretically eventually the server will run out of memory.
This needs to be addressed by introducing some kind of expiration mechanism and cleaning up expired sessions.
* The user interface for selecting sessions and entering PIN numbers use a different way of interacting (HTML forms) with the server than the buttons that
invoke various diagnostics. Perhaps this can be changed with a bit more javascript.
* As mentioned above, the generation of session PIN is currently performed using a non-secure random number genetator, which is convinient for testing
(because the URL one needs to pass to `erigon support` stays the same), but for real-life use, this is not good. A simple improvement could be the default
behaviour being secure random number generator, with an option to use insecure (for example, `--insecure`) to keep it convinient for testing.
* Retrieving command line arguments is only useful if the erigon node is not launched using configuration file. If configutation file is used, then
most of the settings are still not visible to the operator. A possible improvement (which involves also changes in Erigon itself) is to either provide
access to the configutation file, or somehow give access to the "effective" launch settings (i.e. after the configuration file is parsed and applied).
