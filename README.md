# Restrictions Pin Finder

Small app to attempt to find the restrictions PIN for an iOS device by brute force.

Written after the PIN was forgotten for a kid's device and wiping it would of been more work
than writing this little program.

## Usage

Backup the device using iTunes on a desktop computer and then find the folder on the computer
containing the backed up files.

On Mac it will be in the home directory as /Library/Application Support/MobileSync/Backup/<something>
eg.

```
/Users/johndoe//Library/Application Support/MobileSync/Backup/51957b68226dbc9f59cb5797532afd906ba0a1f8
```

Use whatever directory is the latest as the argument to pindecoder:

```
$ pindecoder /Users/johndoe//Library/Application Support/MobileSync/Backup/51957b68226dbc9f59cb5797532afd906ba0a1f8
```

The program will find the plist containing the hashed version of the PIN and will then find
the PIN that matches that hash (which can then be used with your device).
It shouldn't take more than 30 seconds to run.


### Compiling this program

1. [Download and install Go](https://golang.org/doc/install)
2. run go install github.com/gwatts/pindecode

### Other resources

Inspired by information found here:
https://nbalkota.wordpress.com/2014/04/05/recover-your-forgotten-ios-7-restrictions-pin-code/
