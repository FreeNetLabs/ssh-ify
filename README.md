# ssh-ify

A simple SSH tunnel proxy server with user management and WebSocket support.

## Features
- SSH Websocket tunnel proxy
- Simple user management
- Password authentication
- Works with SSH clients like HTTP Injector, DarkTunnel, and Tunn

## Installation

```sh
go install github.com/ayanrajpoot10/ssh-ify@latest
```

## Usage

Configuration is now file based (`config.json`) + optional env vars.

### Start the server
```sh
./ssh-ify
```

### Expected config file
`~/.config/ssh-ify/config.json` (default)

```json
{
  "listen_address": "0.0.0.0",
  "listen_port": 80,
  "ssh_host_key_path": "host_key",
  "users": [
    { "username": "alice", "password": "s3cret" }
  ]
}
```

### Optional env-based user config
- `SSH_IFY_USERS=user1:pass1,user2:pass2`
- config file values are merged with env values


## License
This project is licensed under the [MIT License](LICENSE).
