# Quickstart

## Pre-requisites
- Golang installed locally
- Erigon Node running locally
- Ngrok

## Ngrok install
https://ngrok.com/docs/getting-started/

## Golang install
https://go.dev/doc/install

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

Run the Node
```
./build/bin/erigon --datadir <data_directory> --chain sepolia --metrics
```

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

## Connect the Erigon Node to the Diagnostics System setup
#### Step 1: 
 The app's diagnostic user interface (UI) will automatically serve and open at DIAGNOSTICS_ADDRESS (default is: http://localhost:8080) after you run one of the following commands:

```
  make run-self-signed
```

if you want to connect to remote Erigon node you need to tunnel localhost:8080 using ngrok
```
    ngrok http http://localhost:8080
```
After it you'll get DIAGNOSTICS_ADDRESS, something like (https://84c5df474.ngrok-free.dev) 

#### Step 2: 
Open UI at DIAGNOSTICS_ADDRESS
Follow these steps to create a session:

![create new operation session 1](/_images/create_session_1.png)

press create session

![create new operation session 2](/_images/create_session_2.png)

Enter session name which helps you helassociate session with erigon node user

![create new operation session 3](/_images/create_session_3.png)

#### Step 3: 
Once the new session is successfully created, it will be allocated a unique 8-digit PIN number. You can find this PIN displayed alongside the session in the list of created sessions. Please note that currently, you can only create one session, but support for multiple sessions will be extended in the future.

#### Step 4: 
Go to erigon folder and run command

```
./build/bin/erigon support --debug.addrs localhost:6060 --diagnostics.addr DIAGNOSTICS_ADDRESS --diagnostics.sessions YOUR_SESSION_PIN --insecure
```

Replace `YOUR_SESSION_PIN` with the 8-digit PIN allocated to your session during the previous step. This command will attach the diagnostics tool erigon node using the provided PIN.

#### Step 5: 
Once the diagnostics tool successfully connects to the Erigon node, return to your web browser and reload the page. This step is necessary to query data from the connected node.