# trckr

trckr is a time tracking tool with cross-device syncing support.

## Design philosophy

trckr was designed to accommodate the needs of users with extremely flexible schedules, where tasks don't start and end at particular times of the day, but rather may be started, stopped, and interleaved with other tasks to achieve the target daily hour count for each task.

Furthermore, trckr was designed with multi-device support in mind. The client-server design allows for a server to run on a central computer and to provide an interface to multiple clients. Operations performed on tasks are notified to all clients, thereby providing a synchronization primitive.

## Features

- Starting, stopping and switching between tasks
- Tracking both productive and non-productive periods
- Renaming and deleting tasks
- Cross-device syncing support

## Building

Prerequisites:
- Go >= 1.25
- Make
- C compiler

Fetch the source code:

```bash
git clone http://github.com/idloquy/trckr
```

Build:

```bash
make
```

## Usage

trckr consists of an HTTP server (trckr-http) and a client (trckr-cli). The HTTP server and the client are meant to be run on the same local network or on the same computer, and the client cannot run without an HTTP server.

The server address to be used by trckr-cli can be specified either in the command line or in the configuration file. An example configuration file can be found [here](configs/cli.toml.example) and can be copied to ~/.config/trckr/cli.toml so trckr-cli uses it.

Further usage information can be found in the help messages of the binaries themselves.
