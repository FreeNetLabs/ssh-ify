# ssh-ify

A lightweight, standalone SSH Websocket server written in Go.

## Install

```bash
go install github.com/FreeNetLabs/ssh-ify@latest
```

## Configure

Create a `config.json` in the same directory:

```json
{
  "address": "0.0.0.0",
  "port": 80,
  "banner": "Welcome to ssh-ify!\n",
  "users": [
    {
      "user": "admin",
      "pass": "secret"
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
