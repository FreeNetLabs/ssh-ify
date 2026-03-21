# ssh-ify

A lightweight, standalone SSH server and tunneler written in Go.

## Install

```bash
go install github.com/FreeNetLabs/ssh-ify@latest
```

## Configure

Create a `config.json` in the same directory:

```json
{
  "listen_address": "0.0.0.0",
  "listen_port": 80,
  "banner": "Welcome to ssh-ify!\n",
  "users": [
    {
      "username": "admin",
      "password": "secret"
    }
  ]
}
```

## Run

```bash
ssh-ify
```

## License

See the [LICENSE](LICENSE) file for details.
