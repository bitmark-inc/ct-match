# Pfizer projects

## Pfizer simulator
### Prerequisite

- Go 1.9+

### How to install Go?

#### Install

*For MacOS, using Brew:*
```
$ brew install go
```

If you have Go installed, you can use this command to update Go to latest version:
```
$ brew upgrade go
```

*For Linux or MacOS without using Brew:*
Please referece this guide: https://golang.org/doc/install#install

#### Set Path for Go
*For MacOS, using Brew:*

Add PATH to your .bashrc or .zshrc if you are using ohmyzsh :
```
# .zshrc
# go
export GOROOT=/usr/local/opt/go/libexec
export GOPATH=$HOME/.go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

*For Linux or MacOS without using Brew:*
Please referece this guide: https://github.com/golang/go/wiki/SettingGOPATH

### Build

Create directory for pfizer project:
```
mkdir -p $GOPATH/src/github.com/bitmark-inc/pfizer
```

Change to that directory and checkout the source code:
```
cd $GOPATH/src/github.com/bitmark-inc
git clone git@github.com:bitmark-inc/pfizer.git
```

Change the directory to `flow-simulator` and build the project:
```
cd $GOPATH/src/github.com/bitmark-inc/pfizer/flow-simulator
go build
```

### Configuration

Please use the testnet config file `/flow-simulator/testnet.conf` for your settings.

### Run

```
$ flow-simulator -c testnet.conf
```
