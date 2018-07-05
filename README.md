# Pfizer projects

## Pfizer simulator
### Prerequisite

- Go 1.9+

### Build

Put the project in `$GOPATH/src/github.com/bitmark-inc/pfizer`
Change the directory to `flow-simulator` and run:
```
go build
```

### Configuration

Please use the testnet config file `/flow-simulator/testnet.conf` for your settings.

### Run

```
$ flow-simulator -c testnet.conf
```
