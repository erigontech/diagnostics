# DIAGNOSTICS SYSTEM WEB APPLICATION FOR ERIGON

- [Overview](#overview)
  - [Statement of the problem](#statement-of-the-problem)
  - [Idea of a possible solution](#idea-of-a-possible-solution)
  - [Role for recruiting and onboarding](#role-for-recruiting-and-on-boarding)
- [Development Environment Setup](#development-environment-setup)
  - [Pre-requisites](#pre-requisites)
  - [Erigon Node Setup](#erigon-node-set-up)
  - [Diagnostics Setup](#diagnostics-set-up)
  - [Diagnostics architecture diagram ](#diagnostics-architecture-diagram)
- [How to setup](#how-to-connect-erigon-node-to-the-diagnostics-system)
  - [Local Erigon Node](#local-erigon-node)
  - [Remote Erigon Node](#remote-erigon-node)
    - [Diagnostics setup](#step-1)
    - [Create a session](#step-2)
    - [Retrieve PIN](#step-3)
    - [Connect to node](#step-4)
    - [Observe node data](#step-5)
- [Currently implemented diagnostics](#currently-implemented-diagnostics)
 - [Status Bar](#status-bar)
  - [Current Session](#current-session)
  - [Operating Node](#operating-node)
  - [Switching Between Sessions and Nodes](#switching-between-sessions-and-nodes)
-[Process Tab](#process-tab)
  - [Command](#command)
  - [Flags](#flags)
  - [Node info](#node-info)
- [Network Tab](#network-tab)
  - [Peers data](#peers-data)
  - [Downloader](#downloader)
- [Logs Tab](#logs-tab)
- [Data Tab](#data-tab)
- [Admin Tab](#admin-tab)
- [Available Flags](#available-flags)
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


## Prerequisites
- Go installed locally
- Erigon node running locally (if needed, please review the Erigon node quick setup guide below)


## Erigon Node Set Up
For building the bleeding edge development branch:

```sh
git clone --recurse-submodules https://github.com/ledgerwatch/erigon.git
cd erigon
git checkout main
make erigon
./build/bin/erigon
```

Run the Node. The `<data_directory>` field will be the directory path to your database. The sepolia chain option will allow for quicker setup for testing.

```
./build/bin/erigon --datadir <data_directory> --chain sepolia
```

For more details check [Erigon documentation](https://github.com/ledgerwatch/erigon?tab=readme-ov-file#getting-started)

To set and use a custom address and port, here a
[link to more information on this step](#how-to-run-an-erigon-node-that-can-be-connected-to-the-diagnostics-system)

## Diagnostics Set Up
```
git clone https://github.com/ledgerwatch/diagnostics.git
cd diagnostics
make build
```

Run the application.
```
make run-self-signed
```
[Available flags](#available-flags)

# Diagnostics architecture diagram
![overview](/_diagrams/remote_connection.png)

# How to connect Erigon node to the diagnostics system
## Local Erigon node:
  //TODO: create diag commands readme
## Remote Erigon node: 
#### Step 1: 
[Diagnostics setup](#diagnostics-set-up)

#### Step 2:
Open UI application in browser, you will find the link to it in console.

Follow these steps to create a session:

![create new operation session 1](/_images/create_session_1.png)

Press "Create Session".

![create new operation session 2](/_images/create_session_2.png)

Enter a session name which helps you associate the session with the Erigon node user.

![create new operation session 3](/_images/create_session_3.png)

#### Step 3:
Once the new session is successfully created, it will be allocated a unique 8-digit PIN number. You can find this PIN displayed alongside the session in the list of created sessions. Please note that currently, you can only create one session, but support for multiple sessions will be extended in the future.

#### Step 4:
Ensure that the Erigon node is already running and run the following command. Two important bits are to pass the proper diagnostics address and session PIN.

- `--diagnostics.addr`: By default, the diagnostics address is localhost:8080. You may [tunnel](https://ngrok.com/docs/getting-started/#step-3-put-your-app-online) it to connect to a remote node, you must specify it for this flag.
- `--diagnostics.sessions`: Place the 8-digit PIN allocated to your session during the previous step. This command will attach the diagnostics tool to the Erigon node using the provided PIN.


```
./build/bin/erigon support <flags>
```
[Support command documentation](https://github.com/ledgerwatch/erigon/tree/main/turbo/app#support)


#### Step 5: 
Once the diagnostics tool successfully connects to the Erigon node, return to your web browser and reload the page. This step is necessary to query data from the connected node.


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
    Within the diagnostics application, the operator has the capability to inspect the command line arguments that were used to launch the Erigon node.

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

  Contains detailed info about erigon node.

![sync_stage](/_images/process/node_info.png)



## Network Tab

The Network tab contains information about network peers and the downloader.

### Peers Data

The diagnostics tools allow you to collect and view essential information about network peers, enabling you to monitor and manage your network connections effectively. This data includes details about active peers, static peers, and total seen peers. The information is presented in a tabular format, and you can access detailed data for each section by clicking on the respective row.

#### Overview Data Table

- **Active Peers:** Displays the number of currently active peers in the network.
- **Static Peers:** Indicates the number of static peers defined in your Erigon configuration.
- **Total Seen Peers:** Shows the total number of peers observed during all diagnostics sessions with the current node.

By clicking on the Active, Static, or Total Seen Peers table, you can access detailed information about each peer. The peer details table includes the following columns:

- **Peer ID:** A unique identifier for the peer.
- **Type:** Indicates whether the peer is static, a bootnode, or dynamic.
- **Status:** Displays the status of the peer, indicating whether it's currently active.
- **Total In:** Shows the total amount of data received over the network.
- **Total Out:** Shows the total amount of data sent over the network.
- **In Speed:** Shows the data receiving speed over the network.
- **Out Speed:** Shows the data sending speed over the network.


![sync_stage](/_images/peers.png)

#### Peer Details Popup

- **Main Info:** Info about peer.

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

### Downloader

This table provides detailed information about the progress, download status, estimated time, and resource allocation for "Snapshots" download.

- **Name**: This represents the name or identifier of the stage.
- **Progress**: This indicates the downloading progress as a percentage.
- **Downloaded**: This shows the amount of data that has been downloaded.
- **Total**: The overall size of data for this specific stage.
- **Time Left**: This represents the estimated time remaining for this stage to complete.
- **Total Time**: The total time duration during which this stage has been downloading.
- **Download Rate**: The current download speed for this stage.
- **Upload Rate**: The current upload speed for this stage.
- **Peers**: The number of peers or network nodes involved in this stage of the task.
- **Files**: The total number of files associated with this stage.
- **Connections**: The total number of network connections established for this stage.
- **Alloc**: The amount of resources allocated for this stage.
- **Sys**: The system's resource usage for this stage.

This page also displays all flag values related to the downloader, which helps to understand the current setup and see available tuning options.


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

## Available Flags

The following flags can be used to configure various parameters of the diagnostics UI:

### Configuration File:

- `--config` : Specify a configuration file (default is $HOME/.cobra.yaml).

### Network Settings:

- `--addr` : Network interface to listen on (default is localhost).
- `--port` : Port to listen on (default is 8080).

### Session Management:

- `--node.sessions` : Maximum number of node sessions to allow (default is 5000).
- `--ui.sessions` : Maximum number of UI sessions to allow (default is 5000).

### Logging Configuration:

- `--log.dir.path` : Directory path to store log data (default is ./logs).
- `--log.file.name` : Name of the log file (default is diagnostics.log).
- `--log.file.size.max` : Maximum size of log file in megabytes (default is 100).
- `--log.file.age.max` : Maximum age in days a log file can persist in the system (default is 28).
- `--log.max.backup` : Maximum number of log files that can persist (default is 5).
- `--log.compress` : Whether to compress historical log files (default is false).


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
