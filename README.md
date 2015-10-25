# Restrictions Pin Finder

[![Build Status](https://travis-ci.org/gwatts/pinfinder.svg?branch=master)](https://travis-ci.org/gwatts/pinfinder)

pinfinder is a small application which attempts to to find the restrictions PIN/passcode
for an iOS device by brute force examination of its iTunes backup.

It was written after the PIN was forgotten for a kid's device and wiping it 
would of been more work than writing this little program.

**NOTE**: This program will **not** help you unlock a locked device - It can only help recover the restrictoins
passcode as found in `Settings -> General -> Restrictions`.  More information about Restrictions
can be found [at Apple's web site](https://support.apple.com/en-us/HT201304).

## Download

Binaries for Linux, Mac and Windows can be found at the
[latest releases](https://github.com/gwatts/pinfinder/releases) page.

## Usage

Operating-specifc instructions are below.  In most cases, simply running the program (working around
OS specific security restrictions) should deliver the right result.  Take a look at the Troubleshooting
section if you run into issues.

### Windows

1.  Backup the device using iTunes on a desktop computer.
NOTE: The "encrypt backup" option must be disabled in iTunes.
2. Download pinfinder from the [latest releases](https://github.com/gwatts/pinfinder/releases) page.
3. Select "Open" when prompted by the web browser
4. Drag `pinfinder` from the .zip file to the Desktop
5. Right-click on the start button, and select `Command Prompt`
6. Drag the `pinfinder` icon from the Desktop to the command prompt window, and press return to run it.

![Windows screen grab demo](docs/windows-demo.gif)

_[click here for full size version of above image](https://raw.githubusercontent.com/gwatts/pinfinder/giftest/docs/windows-demo.gif)_


### Mac


1.  Backup the device using iTunes on a desktop computer.
NOTE: The "encrypt backup" option must be disabled in iTunes.
2. Download pinfinder from the [latest releases](https://github.com/gwatts/pinfinder/releases) page.
3. Select the tar.gz file in the download list to open it
4. Right-click on pinfinder and select `Open With` -> `Terminal` - You will receive a warning about the program 
being written by an unknown developer, which you'll need to accept to use it.


![mac screen grab demo](docs/mac-demo.gif)

_[click here for full size version of above image](https://raw.githubusercontent.com/gwatts/pinfinder/giftest/docs/mac-demo.gif)_

### Linux

Download, extract and run the binary.


```
$ ./pinfinder
Searching backup at /Users/johndoe/Library/Application Support/MobileSync/Backup/9afaaa65041cb570cd393b710f392c8220f2f20e
Finding PIN... FOUND!
PIN number is: 1234 (found in 761.7ms)
```

## Troubleshooting

If you have multiple devices or backups, you can pass the exact path to the backup folder to
pinfinder, rather than have it try to find it by itself:

On Mac it will be in the home directory as /Library/Application Support/MobileSync/Backup/<something>
eg.

```
/Users/johndoe/Library/Application Support/MobileSync/Backup/51957b68226dbc9f59cb5797532afd906ba0a1f8
```

On Windows Vista or later it will be something like:

```
\Users\John Doe\AppData\Roaming\Apple Computer\MobileSync\Backup
```

Use whatever directory is the latest as the argument to pindecoder:

```
$ pindecoder /Users/johndoe//Library/Application Support/MobileSync/Backup/51957b68226dbc9f59cb5797532afd906ba0a1f8
```

The program will find the plist containing the hashed version of the PIN and will then find
the PIN that matches that hash (which can then be used with your device).
It shouldn't take more than a few seconds to run.

If the program fails to find the passcode for your device, and you're sure it's searching the right
backup, please [open an issue](https://github.com/gwatts/pinfinder/issues) and copy and paste
the text the program prints in the issue so I can help.



## Compiling this program

If you don't want to use one of the [pre-compiled binaries](https://github.com/gwatts/pinfinder/releases)
you can compile it yourself.

1. [Download and install Go](https://golang.org/doc/install)
2. run go install github.com/gwatts/pindecode

## Other resources

Inspired by information found here:

https://nbalkota.wordpress.com/2014/04/05/recover-your-forgotten-ios-7-restrictions-pin-code/


## Other Notes

Last tested with iOS 8 through 9.1 on OS X 10.10 and Windows 8 with iTunes 12.3
