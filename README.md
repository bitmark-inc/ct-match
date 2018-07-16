# Pfizer projects

## Pfizer simulator
### Prerequisite

- Go 1.9+

### How to install Go?

#### Setup Go Using Brew

```
$ brew install go
```

If you have Go installed, you can use this command to update Go to latest version:
```
$ brew upgrade go
```

#### Set Path for Go
If all process finish, just add PATH to your .bashrc or .zshrc if you are using ohmyzsh :
```
# .zshrc
# go
export GOROOT=/usr/local/opt/go/libexec
export GOPATH=$HOME/.go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

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
