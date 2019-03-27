# Pfizer projects

## Pfizer simulator
### Prerequisite

- Go 1.9+
- Go dep (https://golang.github.io/dep/)

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

#### Install dependencies
*For MacOS, using Brew:*

``` bash
$ brew install dep
$ brew upgrade dep
```

*For Linux
``` bash
sudo apt-get install go-dep
```

On project folder, enter this command:
``` bash
dep ensure
```

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

Build the project:
``` bash
$ cd $GOPATH/src/github.com/bitmark-inc/pfizer
$ go build
```

### Configuration

Please use the testnet config file `testnet.conf` for your settings.

Sample config file:
```hcl
network = "testnet" # bitmark network to run

api_token = "" # Bitmark SDK's API token (see https://sdk-docs.bitmark.com)

wait_time = 10 # waiting time for each step (for demo)

matchingService {
    accounts = [
        {
            identity = "Matching Service 1",
            seed = "9J87CqSvmk7doU5XkpwjnaT7NM85Py6pD"
        },
        {
            identity = "Matching Service 2",
            seed = "9J876x77doS1kJZp4dAMHvXP22KwXouct"
        },
         {
             identity = "Matching Service 3",
             seed = "9J874rk23TJGiLv7Kf6uJoBGZt5DwDCBY"
         },
         {
             identity = "Matching Service 4",
             seed = "9J87B7PKvHccJggdvNuMuNDzoBdB4yqpg"
         },
        {
            identity = "Matching Service 5",
            seed = "9J873bK5yFWvAZgoKtWcvR2LbETJDyua8"
        }
    ] # pre-defined accounts for matching services
    select_asset_prob = 0.3 # probability of selecting assets from sponsors to issue more and send to participants 
    match_prob = 0.4 # probability of selecting a participant for a specific trial
    match_data_approval_prob = 0.7 # probability of approving a trial on evaluation (after receiving from participant)
}

sponsors {
    accounts = [
        {
            identity = "Stanford University",
            seed = "9J87E888gSWzSE5bo6Pw62aNH8X52Y799"
        },
        {
            identity = "University of California",
            seed = "9J878GN96ArWWdqjDmqruPMsA8o1VRtht"
        },
        {
            identity = "Noah Merin Los Angeles",
            seed = "9J875vioxDXnZ43ft4qisUS3PebucqjDo"
        },
         {
             identity = "WCCT Cypress",
             seed = "9J878sSgsFW1RMaT18N5bfF2JZ5GzTAwK"
         },
        {
            identity = "Stanford Cancer Institute",
            seed = "9J874S1M2kv5PEwaUX7aQYTor4LXYEk8L"
        },
        {
            identity = "ProSciento Inc.",
            seed = "9J87CNq39EfHFUfRZ8xPvBAYm6YSvVAVw"
        },
        {
            identity = "Cedars Sinai Los Angeles",
            seed = "9J877CfjEp1pJBfL7CrQBzSfbBCHeh1YW"
        },
        {
            identity = "Adam Schickedanz",
            seed = "9J87ESXqsYm4Upr8GvjirckxHJdiGJAEp"
        },{
            identity = "UCSF School of Dentistry",
            seed = "9J876GhEZE5v6FLYLcKwNjhZwGSDkHUWB"
        }
    ] # pre-defined accounts for sponsors
    sponsor_data_approval_prob = 0.7 # probability of approving trials which is sent from matching services after evaluation
    trials_per_sponsor_min = 2 # minimum number of trials to issue for each sponsor
    trials_per_sponsor_max = 3 # maximum number of trials to issue for each sponsor
    studies_pool = [
        "Bisphenol A and Muscle Insulin Sensitivity",
        "Gas Exchange Kinetics and Work Load During Exercise",
        "Improving Islet Transplantation Outcomes With Gastrin",
        "HostDx Sepsis in Patients With Acute Respiratory Infections",
        "Energy Devices for Rejuvenation",
        "High School Start Time and Teen Migraine Frequency",
        "The Natural History of Danon Disease",
        "Restylane Silk Microinjections to Cheeks",
        "iBeat Wristwatch Validation Study",
        "Glucose Control Using 1,5-AG Testing",
        "Cut Your Blood Pressure 3",
        "18F-Fluorocholine for the Detection of Parathyroid Adenomas",
        "Efficacy and Safety of SYN-010 in IBS-C",
        "Postpartum Care Timing: A Randomized Trial",
        "Cardiac Recovery Through Dietary Support",
        "Ford Rumination and Mindfulness Merit",
        "Effects of Playing Pokemon Go on Physical Activity",
        "Mobile Virtual Positive Experiences for Anhedonia",
        "Behavioral Family Therapy and Type One Diabetes",
        "Sun Safety Skills for Elementary School Students"
    ] # Studies that the app will pick randomly to name the trial
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
$ ./pfizer -c testnet.conf
```
