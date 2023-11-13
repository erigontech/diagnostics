# DIAGNOSTICS SYSTEM WEB APPLICATION FOR ERIGON

- [Overview](#overview)
  - [Statement of the problem](#statement-of-the-problem)
  - [Idea of a possible solution](#idea-of-a-possible-solution)
  - [Role for recruiting and onboarding](#role-for-recruiting-and-on-boarding)
- [Development Environment Setup](#development-environment-setup)
  - [Pre-requisites](#pre-requisites)
  - [Erigon Node Setup](#erigon-node-set-up)
  - [Diagnostics Setup](#diagnostics-set-up)
  - [Connect Erigon The Diagnostics System](#connect-the-erigon-node-to-the-diagnostics-system-setup)
- [Architecture of diagnostics system](#architecture-of-diagnostics-system)
- [Currently implemented diagnostics](#currently-implemented-diagnostics)
  - [Code version](#code-version)
  - [Command line arguments](#command-line-arguments)
  - [Logs](#logs)
  - [Reorg scanner](#reorg-scanner)
  - [Sync stages](#sync-stages)
  - [Block body download](#block-body-download)
- [Ideas for Possible improvements](#ideas-for-possible-improvements)

# Overview

## Statement of the problem

According to our estimations, Erigon is used by hundreds of people. Some users are individuals, others - companies that use it for internal purposes, and there are also companies that "sell" access to their Erigon nodes to other individuals and companies.
Very often, users encounter issues when working with Erigon. We can classify them roughly in the following categories by their cause:
1. Caused by the mismatching expectation. User expects Erigon to do something that it cannot do.
2. Caused by accidental user errors (typos in the command line, for example).
3. Caused by underlying software or hardware issues on user’s system (for example, running out of disk space, faulty memory, bug in operating system).
4. Caused by bugs in Erigon.
5. Caused by bugs in libraries that Erigon uses.
6. Caused by other systems that are used in conjunction with Erigon (for example, Consensus Layer implementations).

This classification should be viewed as "work in progress" and we will refine it as we move on with the project.
As we communicate with the users about these issues, we would like to first determine whether it is an issue of type (1), (2), (3), or (4), and so on, or perhaps a new kind of issue. Some issues, for example, of type (4), which are the most “interesting” for Erigon developers, can be further classified.
Often, it is enough to ask the user to show error message or console output or log file to identify the cause. Often the issues are easy to reproduce if developers have the same type of node that user has.
But there are many cases where even looking at the console output is not enough to diagnose the issue. Moreover, the issue may be transient and disappear when the user tries to restart the system with some additional tracing code.
Our prediction is that as the complexity of Erigon as a system grows, the diagnostics of user issues will pose bigger and bigger challenge unless it is addressed in a systematic way.

## Idea of a possible solution

We do not know if this possible solution will be workable, but we should try. In order to address the problem described above in a systematic way, we need to create a system and/or a process for dealing with user issues, i.e. to diagnose their cause.
Diagnostics require gathering information about the user’s Erigon node. Some information should be gathered in any case, because it is universally useful for investigating the cause of most issues:
Version of Erigon.
Basic parameters of the user’s systems and its hardware, for example, available and used memory, available disk space.
Snippet of the recent console output.

Further investigation often requires more data, and probing for such data is usually done in an interactive manner, in order to optimise for the amount of communication. This applies both to human communication (when diagnostics is done in our current way), and to the communications between diagnostics system and user’s Erigon node.
Diagnostics may be performed by the diagnostics system in the manual mode, where probing for more data is done by the human operator, who uses the diagnostics system to visualise the data received before, and decides what else needs to be seen. This would be initial mode of operation for the diagnostics system, as we start to learn what capabilities it should have.
It is conceivable that as the diagnostics system develops, it may be able to perform diagnostics automatically using some pattern matching and heuristics.

## Role for recruiting and on-boarding

Since the diagnostics system is an application which is only very loosely connected with Erigon, and so far is more self-contained, learning it and working with it should require much
less time than learning to work with the Erigon code.
Also, working with the diagnostics system requires understanding how Erigon is run and operated by the user, and hopefully should also give some exposure to what kind of issues
usually occur and what are useful diagnostics to deal with them.

Therefore, we think it can be very useful to ask for contributions (improvements) to the diagnostics system as a part of our recruiting process (to pre-screen the candidates who are
interested and capable), and also as a part of on-boarding for the developers who have recently joined the team. For worthy improvements, bounties can be paid, since we do not
necessarily want people to work for free.

# Development Environment Setup


## Pre-requisites
- Golang installed locally
- Erigon Node running locally (if needed, please review the Erigon Node quick setup guide below)


## Erigon Node Set Up
Clone the Erigon repository
```
git clone --recurse-submodules -j8 https://github.com/ledgerwatch/erigon.git
```

Change into the repo, and make sure you are on the ```devel``` branch
```
cd erigon
git checkout devel
```

Build the repo
```
make erigon
```

Run the Node. To make sure that it connects to the diagnostics system, add the --metrics flag.
The `<data_directory>` field will be the directory path to your database. The sepolia chain and the --internalcl options will allow for quicker setup for testing

```
./build/bin/erigon --datadir <data_directory> --chain sepolia --internalcl --metrics
```

Check the prometheus logs by navigating to the url below
```
http://localhost:6060/debug/metrics/prometheus
```
To set and use a custom address and port, here a
[link to more information on this step](#how-to-run-an-erigon-node-that-can-be-connected-to-the-diagnostics-system)

## Diagnostics Set Up
```
git clone https://github.com/ledgerwatch/diagnostics.git
cd diagnostics
make build
```

Run the application. This may take a while. Expect to see a TLS Handshake error in the terminal
```
make run-self-signed
```

To view the application in your browser, go to the URL `https://localhost:8080/ui`. Your browser will likely ask to accept the risks (due to self-signed certificate), do that.

[Link to more information on this step](#how-to-build-and-run)

## Connect the Erigon Node to the Diagnostics System setup
[Link to more information on this step](#how-to-connect-erigon-node-to-the-diagnostics-system)

# Testing & Linting

Running the tests including integration and end to end can be done using the Makefile:
```
make test
```

Linting with golangci:
```
  make lint
```


# How to run an Erigon node that can be connected to the diagnostics system

For an Erigon node to be connected to the diagnostics system, it needs to expose metrics using this command line flag:

```
--metrics
```

By default, the metrics are exposed on `localhost` and port `6060`. In order to expose them on a different networking interface and/or different port,
the following command line flags can be used:

```
--metrics.addr <IP of interface> --metrics.port <port>
```

If you are not sure what kind of node to run, you can try Ethereum Sepolia testnet with Caplin (Erigon's experimental embedded Consensus Layer support).
Caplin works on Sepolia in `devel` branch of Erigon, but this wll be included into the next release. Build Erigon from `devel` branch, choose data directory
and run this command:

```
erigon --datadir <data_directory> --chain sepolia --internalcl --metrics
```

The flag `--internalcl` enables Caplin, which means that you won't need to install a Consensus Layer separately, and this will make your work simpler.

In order to check is the metrics are exposed on given interface or port, you can check it in the browser by going to
```
http://<metrics.addr>:<metrics.port>/debug/metrics/prometheus
```

If metrics are exposed, textual representation of metrics will be displayed in the browser.

# How to connect Erigon node to the diagnostics system
![diagnostics system connection](/_images/diagnostics-connection-anim.gif)
#### Step 1: 
 The app's diagnostic user interface (UI) will automatically open at https://localhost:8080 after you run one of the following commands:
```
./_bin/diagnostics --tls.cert demo-tls/diagnostics.crt --tls.key demo-tls/diagnostics-key.pem --tls.cacerts demo-tls/CA-cert.pem
```
or
```
  make run-self-signed
```


#### Step 2: 
Please note that you may need to accept the security risks associated with the self-signed certificate. This action is required only the first time you access this URL, as your browser will remember your choice for subsequent visits.

#### Step 3: 
Follow these steps to create a session:

![create new operation session 1](/_images/create_session_1.png)

press create session

![create new operation session 2](/_images/create_session_2.png)

Enter session name which helps you helassociate session with erigon node user

![create new operation session 3](/_images/create_session_3.png)

#### Step 4: 
Once the new session is successfully created, it will be allocated a unique 8-digit PIN number. You can find this PIN displayed alongside the session in the list of created sessions. Please note that currently, you can only create one session, but support for multiple sessions will be extended in the future.

#### Step 5: 
Ensure that the Erigon node is already running on your system. Then, open a new console window and run the following command. Be sure to specify the session PIN at the end of the `--diagnostics.url` command line flag. Since the website uses a self-signed certificate without a properly allocated CName, you need to use the `--insecure` flag to establish a connection.

```
./build/bin/erigon support --debug.urls http://localhost:6060 --diagnostics.url https://localhost:8080 --diagnostics.sessions YOUR_SESSION_PIN --insecure
```

Replace `YOUR_SESSION_PIN` with the 8-digit PIN allocated to your session during the previous step. This command will attach the diagnostics tool erigon node using the provided PIN.

#### Step 6: 
Once the diagnostics tool successfully connects to the Erigon node, return to your web browser and reload the page. This step is necessary to query data from the connected node.

#### Step 7: 
Currently diagnostics UI support "Process", "Logs", "Data" tabs
# Architecture of diagnostics system

Following diagram shows schematically how the process of diagnostics works. Erigon nodes that can be diagnosed, need to be running with `--metrics` flag.
Diagnostics system (HTTP/2 website) needs to be running somewhere. For the public use, it can be a website managed by Erigon team, for example. For
personal and testing use, this can be locally run website with self-signed certificates.

In order to connect Erigon node to the Diagnostics system, user needs to start a process with a command `erigon support`, as described earlier.
The initiations of network connections are shown as solid single arrows. One can see that `erigon support` initiates connections to both Erigon node
and the diagnostics system. Then, it uses the feature of HTTP/2 called "duplex connection", to create a logical tunnel that allows diagnostics system
to make HTTP requests to the Erigon node, and receive the information exposed on the metrics port. The URLs used to connect `erigon support` to the
diagnostics system, start with `/support/` prefix, followed by the PIN of the session. In the code inside `cmd/root.go`, this corresponds to the
`BridgeHandler` type.

Operators (those who are trying to assist the Erigon node users) also access Diagnostics system, but in the form of User Interface, built using HTML
and Javascript. The URLs used for such access, start with `ui/` prefix. In the code inside `cmd/root.go`, this corresponds to the `APIHandler` type.

![diagnostics system architecture](/_images/diagnostics.drawio.png)

# Currently implemented diagnostics

## Status Bar
  The Status Bar in our application displays essential information about the current session and operating erigon node. It also allows operators to switch between different sessions and nodes quickly.
  ![status-bar](/_images/statusbar/status_bar.png)
  - ### Current Session
    The "Current Session" section of the status bar displays information about the currently active session.
    ![status-bar-session](/_images/statusbar/status_bar_sessions.png)

  - ### Operating Node
    The "Operating Node" section of the status bar displays information about the currently active erigon node.
    ![status-bar-node](/_images/statusbar/status_bar_nodes.png)

  - ### Switching Between Sessions and Nodes
    You can easily switch between different sessions and nodes by clicking on the respective session or node in the status bar.

  - ### Session/Node Popup
    After clicking on a session or node, a popup will appear, displaying a list of available sessions or nodes to switch between.

    To Switch Between Sessions or Nodes:
    - Click on the "Current Session" or "Operating Node" section of the status bar.
    - A popup window will appear, listing all available sessions or nodes.
    - Click on the desired session or node in the list.
    - The application will switch to the selected session or node.

    ![sessions-popup](/_images/statusbar/sessions_popup.png)
    ![nodes-popup](/_images/statusbar/nodes_popup.png)
## Process Tab
  - ### Command:
    Within the diagnostics application, the operator has the capability to inspect the command line arguments that were used to launch the Erigon node. This functionality is implemented in the file `internal/erigon_node/erigon_client.go`, specifically within the function named `CMDLineArgs`. 

![cmd line](/_images/process/cmd_line.png)

  - ### Flags:
    In the Erigon node, the operator can examine the flags that are set in the CLI context by the user when launching the node. This functionality is implemented in the file internal/erigon_node/erigon_client.go. The relevant code for retrieving and displaying these flags can be found within this file.

    This feature is especially valuable when the user launches Erigon using a configuration file with the --config flag, as command line arguments alone may not fully capture the true state of the launch settings. The flags returned represent the result after parsing both the command line arguments and the configuration file, providing a comprehensive view of the configuration used to start the Erigon node.

    The flags table within the diagnostics application typically includes three columns:
    - Name: This column contains the name of the flag, representing the specific configuration option or setting.
    - Value: The value column displays the current value of the flag, indicating the configuration set for that particular option.
    - Default: The default column provides information on whether the flag was set by the user. It shows "false" if the flag was set by the user and "true" if the flag is using its default value.
    
    This table allows the operator to easily review and understand the configuration settings for various flags in the Erigon node, including whether these settings were user-defined or are using default values.

![flags](/_images/process/flags.png)

- ### Node info

  Contains detailed info about erigon node. Coresponding code located at `internal/api/ui_handler.go`

![sync_stage](/_images/process/node_info.png)

- ### Sync stages

  This is another example of how the diagnostics system can access the Erigon node's database remotely, via `erigon support` tunnel.
  This feature adds an ability to see the node's sync stage, by returning the number of synced blocks per stage.

![sync_stage](/_images/process/sync_stages.png)

- ### Reorgs

  This is the first very crude example of how diagnostics system can access Erigon node's database remotely, via erigon support tunnel. Re-orgs can be identified by the presence of multiple block headers with the same block height but different block hashes.

  One of the ideas for the further development of the diagnostics system is the addition of many more such useful "diagnostics scripts", that could be run against Erigon's node's database, to check the state of the node, or certain inconsistencies etc.

  The corresponding code in Erigon is in the file internal/sessions/cache.go, and it relies on a feature recently added to the Erigon's code, which is mdbx.PathDbMap(), the global function that returns the mapping of all currently open MDBX environments (databases), keyed by the paths to their directories in the filesystem. This allows cache.go to create a read-only transaction for any of these environments (databases) and provide remote reading by the diagnostics system.

![sync_stage](/_images/process/find_reorgs.png)

## Network Tab

1. ### Peers data

Our diagnostics tools allow you to collect and view essential information about network peers, enabling you to monitor and manage your network connections effectively. This data includes details about active peers, static peers, total seen peers, and any encountered errors. The information is presented in a tabular format, and you can access detailed data for each section by clicking on the respective row.

#### Overview Data Table:

- **Active Peers:** Displays the number of currently active peers in the network.
- **Static Peers:** Indicates the number of static peers defined in your Erigon configuration.
- **Total Seen Peers:** Shows the total number of peers observed during all diagnostics sessions with the current node.
- **Total Errors:** Provides information about any errors or issues encountered with network peers.

By clicking on the Active, Static, or Total Seen Peers table, you can access detailed information about each peer. The peer details table includes the following columns:

- **Peer ID:** A unique identifier for the peer.
- **Type:** Indicates whether the peer is static, a bootnode, or dynamic.
- **Status:** Displays the status of the peer, indicating whether it's currently active.
- **Network Usage:** Shows the total amount of data sent and received over the network.
- **Errors Count:** Shows the total count of errors encountered with this peer.

This detailed peer information allows you to gain insights into your network connections, helping you troubleshoot and manage your network effectively.

![sync_stage](/_images/peers.png)

#### Peer Details Popup

**Main Info:**

- **ID:** A unique identifier for the network peer.
- **Bytes In:** The total amount of data received by the peer, measured in megabytes (MB).
- **Bytes Out:** The total amount of data sent by the peer, measured in kilobytes (KB).

**Network Usage By Capability:**

- **Type:** The capability of the communication channel.
- **Bytes In:** The amount of data received from the peer.
- **Bytes Out:** The amount of data sent to the peer.

**Network Usage By Type:**

- **Type:** The specific message type. (https://github.com/ethereum/devp2p/blob/master/caps/eth.md#protocol-messages)
- **Bytes In:** The amount of data received for each type.
- **Bytes Out:** The amount of data sent for each type.

Use this detailed popup to analyze and monitor the network activity of peers, aiding in troubleshooting, optimizing performance, and resource allocation. It offers a clear and concise breakdown of peer interactions with the network, facilitating better network management and decision-making.

![sync_stage](/_images/peer_details.png)

### Snapshots data
This table provides detailed information about the progress, download status, estimated time, and resource allocation for "Snapshots" stages

- **Part**: stage Name - This represents the name or identifier of the stage.
- **Progress**: Download Percentage - This indicates the downloading progress as a percentage.
- **Downloaded**: Downloaded Data - This shows the amount of data that has been downloaded.
- **Total**: Total Data Size - The overall size of data for this specific stage.
- **Time Left**: Estimated Time Remaining - This represents the estimated time remaining for this stage to complete.
- **Total Time**: Total Elapsed Time - The total time duration during which this stage has been downloading.
- **Download Rate**: Current Download Speed - The current download speed for this stage.
- **Upload Rate**: Current Upload Speed - The current upload speed for this stage.
- **Peers**: Peer Count - The number of peers or network nodes involved in this stage of the task.
- **Files**: Number of Files - The total number of files associated with this stage.
- **Connections**: Network Connections - The total number of network connections established for this stage.
- **Alloc**: Allocated Resources - The amount of resources allocated for this stage.
- **Sys**: System Resource Usage - The system's resource usage for this stage.

Background change it's color as soon as part downloaded
![snapshot](/_images/snapshot_sync.png)

## Logs Tab

Since version 2.43.0, Erigon nodes write logs by default with `INFO` level into `<datadir>/logs` directory, there is log rotation. Using diagnostics system,
these logs can be looked at and downloaded to the operator's computer. Viewing the logs is one of the most frequent requests of the operator to the user,
and it makes sense to make this process much more convenient and efficient. The corresponding code in Erigon is in the file `internal/erigon_node/erigon_client.go`.
Note that the codes does not give access to any other files in the file system, only to the directory dedicated to the logs.

![logs](/_images/logs.png)

## Data Tab
Operator has the capability to inspect the databases and their tables. This functionality is implemented in the file  `internal/erigon_node/remote_db.go`.

![flags](/_images/dbs.png)

## Admin Tab
### Session Management

- **List Sessions**: View active sessions.

- **Create New Session**: Start a new session.

- **Obtain PIN for Session**: Generate a session PIN for security.

### Data Management

- **Clear All Data**: Permanently delete all sessions and data. Use with caution:

  - Dignostic updates may contain breaking changes which will result in crashes.  To prevent application crashes need to cleard data.

**Note:** Data deletion is irreversible.


![flags](/_images/dbs.png)

## (--DEPRECATED--) Header Download
This is another crude example of monitoring an algorithm involving many items transitoning through series of states. On the erigon side, the code is spread across `dataflow/stages.go` and `diagnostics/header_downloader_stats.go`. The parameters considered for monitoring are decided based on header download states used in `turbo/stages/headerdownload/header_algos.go` and `eth/stagedsync/stage_headers.go`.
The header downloader algorithm on the diagnostics system side is stored in `headers_download.go` file. The code in the file is reused from the `bodies_download.go` file which contains the code for fetching the bodies download state from erigon.

## (--DEPRECATED--) Block Body Download

This is the first crude example of monitoring an algorithms involving many items (in that case block bodies) transitioning through the series of states.
On the Erigon side, the code is spread across files `dataflow/stages.go`, where the states of each block body in the downloading algorithm are listed,
and the structure `States` is described. This structure allows the body downloader algorithm (in the files `eth/stagedsync/stage_bodies.go` and
`turbo/stages/bodydownload/body_algos.go`) to invoke `AddChange` to report the change of state for any block number. The structure `States` intends to
have a strict upper bound on memory usage and to be very allocation-light. On the other hand, the function `ChangesSince` is called by the code in
`diagnostics/block_body_download.go` to send the recent history of state changes to the diagnostics system (via logical tunnel of `erigon support` of course).
On the side of the diagnostics system, in the file `cmd/bodies_download.go`, there are two functions. One, `bodies_download` is generating output
HTML representing the current view of some limited number of block bodies being downloaded (1000). This function keeps querying the Erigon node
roughly every second and re-generates the HTML (using template in `assets/template/body_download.hml`). The re-generated HTML is written to, and is
consumed by the javascript function `bodiesDownload` in the `assets/script/session.js`, which keeps replacing the `innerHTML` field in a div element
whenever the new HTML piece is available.
Each state is represented by a distinct colour, with the colour legend is also defined in the template file.

![body download](/_images/body_download.png)

# Ideas for possible improvements

If you are looking at this because you would like to apply to be a part of Erigon development team, the best you can do is to try to first run the
diagnostics system locally as described above, then study the code in the repository and think of a way to improve it. This repository has been
initially created by a person with very little experience in web server development, web design, javascript, and, more crucially, it has been
created in a bit of a rush.
Therefore, there should be a lot of things that can be improved in terms of best practices, more pleasant user interface, code simplicity, etc.

There are some functional improvements that could be quite useful, for example:

* Reorg scanner is very basic and it does not have a concept of a "deep" reorg (deeper than 1 block). For such situations, it will just show the consecutive
  block numbers as all having a reorg. It would be better to aggregate these into deep reorgs, and also perhaps show if there are more than 1 branch at each
  reorg point.
* For the reorg scanner, add the ability to click on the block numbers and get more information about that particular reorg, for example, block producers
  for each of the block participating in the reorg, or difference in terms of transactions.
* Adding more "diagnostics scripts" that remotely read DB to check for the current progress of stages in the staged sync.
* Adding a monitoring for header downloader as well as for body downloader.
* Perhaps embedding some metrics visualisation (have no idea how to do it), since all "prometheus"-style metrics are also available to the diagnostics system?
* Ability to extract and analyse go-routine stack traces from Erigon node. To start with, extract something like `debug/pprof/goroutine?debug=2`, but for Erigon
  this would likely result in a lot of go-routines (thousands) with similar traces related to peer management. Some analysis should group them into cluster of similar
  stack traces and show them as aggregates.
* Add log rotation system similar to what has recently been done for Erigon (using lumberjack library).
