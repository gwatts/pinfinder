# iOS Screen Time & Restrictions Passcode Finder

[![Build Status](https://travis-ci.org/gwatts/pinfinder.svg?branch=master)](https://travis-ci.org/gwatts/pinfinder)

Pinfinder is a small program for Mac, Windows and Linux which attempts to to find the screen time and/or restrictions passcode
for an iOS device (iPhone, iPad or iPod Touch) from a normal backup of the device made by iTunes on your computer.

See https://pinfinder.net/ for quick instructions on how to download and use it.

**NOTE**: This program will **not** help you unlock a locked device - It can only help recover the restrictions
passcode as found in `Settings -> General -> Restrictions`.  More information about Restrictions
can be found [at Apple's web site](https://support.apple.com/en-us/HT201304).


# Compiling this program

(Most people don't need to do any of this; just go to https://pinfinder.net/ instead, unless you're technically inclined to read on ;-) )

If you are running on a platform other than Mac, Windows or Linux you will need to compile the program yourself:


First [Download and install Go](https://golang.org/doc/install).

Once Go is installed, you can clone the pinfinder repo and build it.   Pinfinder uses the new module system found in Go 1.11 and later to track its dependencies.


```bash
cd ~
git clone https://github.com/gwatts/pinfinder.git
cd pinfinder
GO111MODULE=on go build .
./pinfinder
```

## Other resources

Inspired with thanks by information found here:

https://nbalkota.wordpress.com/2014/04/05/recover-your-forgotten-ios-7-restrictions-pin-code/


## Other Notes

Last tested with iOS 8 through 12.2 on OS X 10.10, 10.11, 10.12, 10.12, 10.13 Windows XP and Windows 8 with iTunes 12.7

NOTE: Recovery of an iOS 12 passcode requires an **encrypted** iTunes backup.
