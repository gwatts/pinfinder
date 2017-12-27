# iOS Restrictions Passcode Finder

[![Build Status](https://travis-ci.org/gwatts/pinfinder.svg?branch=master)](https://travis-ci.org/gwatts/pinfinder)

Pinfinder is a small application for Mac, Windows and Linux which attempts to to find the restrictions passcode
for an iOS device (iPhone, iPad or iPod Touch) by brute force examination of its iTunes backup, without needing to jailbreak the device.

For full details on how to use it, see https://pinfinder.net/

**NOTE**: This program will **not** help you unlock a locked device - It can only help recover the restrictions
passcode as found in `Settings -> General -> Restrictions`.  More information about Restrictions
can be found [at Apple's web site](https://support.apple.com/en-us/HT201304).


# Compiling this program

If you are running on a platform other than Mac, Windows or Linux you will need to compile the program yourself:


1. [Download and install Go](https://golang.org/doc/install) - Be sure to follow the instructions to [setup a workspace](https://golang.org/doc/code.html#Workspaces) and set a `GOPATH` environment variable to suit
2. run `go get github.com/gwatts/pinfinder`

If you just want to compile the program as quick as possible, install Go from the web site above, and run the following steps to build and install it to `~/pinfinder/bin/pinfinder`

```bash
cd ~
mkdir ~/pinfinder
cd pinfinder
mkdir src bin pkg
export GOPATH=~/pinfinder
go get github.com/gwatts/pinfinder
bin/pinfinder
```

## Other resources

Inspired with thanks by information found here:

https://nbalkota.wordpress.com/2014/04/05/recover-your-forgotten-ios-7-restrictions-pin-code/


## Other Notes

Last tested with iOS 8 through 11.2.1 on OS X 10.10, 10.11, 10.12 Windows XP and Windows 8 with iTunes 12.7
