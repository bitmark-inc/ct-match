# Pfizer projects

## Pfizer simulator
### Prerequisite

- Go 1.9+

### How to install Go?

#### Install

*For MacOS, using Brew:*
``` bash
$ brew install go
```

If you have Go installed, you can use this command to update Go to latest version:
``` bash
$ brew upgrade go
```

*For Linux or MacOS without using Brew:*
Please referece this guide: https://golang.org/doc/install#install

#### Set Path for Go
*For MacOS, using Brew:*

Add PATH to your .bashrc or .zshrc if you are using ohmyzsh :
``` bash
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
``` bash
$ mkdir -p $GOPATH/src/github.com/bitmark-inc/pfizer
```

Change to that directory
``` bash
$ cd $GOPATH/src/github.com/bitmark-inc
```

Checkout source code via ssh:
``` bash
$ git clone git@github.com:bitmark-inc/pfizer.git
```

Or you can checkout source code via git https:
``` bash
$ git clone https://github.com/bitmark-inc/pfizer.git
```

Change the directory to `flow-simulator` and build the project:
``` bash
$ cd $GOPATH/src/github.com/bitmark-inc/pfizer/flow-simulator
$ go build
```

### Configuration

Please use the testnet config file `/flow-simulator/testnet.conf` for your settings.

Sample config file:
```hcl
network = "testnet" # bitmark network to run

wait_time = 10 # waiting time for each step (for demo)

matchingService {
    accounts = [
        "5XEECtvPGJ84ogn76W5BoPCpqL3TG3zUcScMpvH2zTUu8xeMrXvGUgW",
        "5XEECsk7VkpvHqPdGH4664P9q3sXkLAJEdSUCFq2FBVKn97B3GPPS7V",
        "5XEECsUkdJ7rt9eo2AAGN9qx6pQZ1hi5Hs5pB6mZ5VBwoZmtKbu3Lpp",
        "5XEECsADuedFjWk7HzHSyJx7xF4NXKp7DY2pedJ8DyMefpg6BTDP5D3",
        "5XEECtF9sMSc6gN5Agu7qsUsif1uu3PvjWQN9wvoncJnY3KFt7HASk3",
        "5XEECtLc3AakrabW1PvjoZD92sro6AnSXMG95D5Fa7dwJuDowgFY8WC"
    ] # pre-defined accounts for matching services
    select_asset_prob = 0.3 # probability of selecting assets from sponsors to issue more and send to participants 
    match_prob = 0.4 # probability of selecting a participant for a specific trial
    match_data_approval_prob = 0.7 # probability of approving a trial on evaluation (after receiving from participant)
}

sponsors {
    accounts = [
        "5XEECsftWZMHs1qzpvHxhE1PPYd4eNcXLTfS5M72d1ePehSmEj142HL",
        "5XEECsQJdzzzvR9xSGt52389jQDU8V2ikGmoMqjuEZmYYTStUFHBHzg",
        "5XEECtqNr1daLUVKoCssP5bRmceR5CBmWKTA4SstgCQ7bYQwGiUSBXm",
        "5XEECtZAJKaUBrpXkaap8CdJedy4Sr6XfvahLtNj4rZd2MqqRzTbbnj"
    ] # pre-defined accounts for sponsors
    sponsor_data_approval_prob = 0.7 # probability of approving trials which is sent from matching services after evaluation
    trials_per_sponsor_min = 2 # minimum number of trials to issue for each sponsor
    trials_per_sponsor_max = 3 # maximum number of trials to issue for each sponsor
}

participants {
    participant_num = 20 # number of participants
    participant_accept_match_prob = 0.8 # probability of accepting a trial when receiving from sponsors (final step)
    participant_submit_data_prob = 0.8 # probability of submiting medical data to matching service after receving trial
    participant_accept_trial_invite_prob = 0.8 # probability of accepting trial invitation from matching service
}
```



### Run

``` bash
$ ./flow-simulator -c testnet.conf
```
