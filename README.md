⚠️⚠️⚠️ **WORK IN PROGRESS** ⚠️⚠️⚠️

# tm-events
This repository provides:
- a library for robust Tendermint event subscriptions that can guarantee events won't be missed and subscriptions won't be dropped
- easy query creation and event filters
- a shell tool to wait for specified events before continuing execution

# Build

Install [libzmq](https://zeromq.org/download/). On Mac OS, do
```sh
brew install zmq
```

To build,
```sh
make build
```
