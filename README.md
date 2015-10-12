# Restrictions Pin Finder

[![Build Status](https://travis-ci.org/gwatts/pinfinder.svg?branch=master)](https://travis-ci.org/gwatts/pinfinder)

pinfinder is a small application which attempts to to find the restrictions PIN/passcode
for an iOS device by brute force examination of its iTunes backup.

It was written after the PIN was forgotten for a kid's device and wiping it 
would of been more work than writing this little program.

## Download

Binaries for Linux, Mac and Windows can be found at the
[latest releases](https://github.com/gwatts/pinfinder/releases) page.

## Usage

1.  Backup the device using iTunes on a desktop computer.  NOTE: The "encrypt backup" option 
must be disabled in iTunes.
2. Run pinfinder from a command prompt:
  * __Mac__: Right click on the `pinfinder` program and select "Open with Terminal.app" - You will receive a warning about the program being written by an unknown developer, which you'll need to accept to use it.  Alternatively start `Terminal.app` (located in `Applications->Utilities`) and run the program from there (eg. `Downloads/pinfinder`)
  * __Windows__: Download the zip file and select `Open` - Drag `pinfinder` to your `Downloads` folder.  Right click on the start button and select `Command Prompt` - You should be able to type `Downloads\pinfinder.exe` to run the program.
  * __Linux__: I doubt you need help ;-)


pinfinder will attempt to find the latest backup and extract the restrictions PIN from it.

```
$ ./pinfinder
Searching backup at /Users/johndoe/Library/Application Support/MobileSync/Backup/9afaaa65041cb570cd393b710f392c8220f2f20e
Finding PIN... FOUND!
PIN number is: 1234 (found in 761.7ms)
```

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


### Compiling this program

If you don't want to use one of the [pre-compiled binaries](https://github.com/gwatts/pinfinder/releases)
you can compile it yourself.

1. [Download and install Go](https://golang.org/doc/install)
2. run go install github.com/gwatts/pindecode

### Other resources

Inspired by information found here:

https://nbalkota.wordpress.com/2014/04/05/recover-your-forgotten-ios-7-restrictions-pin-code/


### Other Notes

Last tested with iOS 8 through 9.02 on OS X 10.10 and Windows 8 with iTunes 12.3
