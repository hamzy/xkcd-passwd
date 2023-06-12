# xkcd-passwd
CLI for generating XKCD style passwords written in golang

## Building

```bash
~$ git clone https://github.com/hamzy/xkcd-passwd.git
~$ cd xkcd-passwd/
~/xkcd-passwd$ ./build-dictionary1.sh  # or ./build-dictionary2.sh
```

## Testing built program

```bash
~/xkcd-passwd$ ./test-xkcd-defaults.sh 
```

## Sample defaults.json

Examples of differing options are in the files xkcd-defaults*.json

The program `xkcd-password` needs a defaults file located in either
in the home directory (`~/defaults.json`) or in the current directory
(`defaults.json`).

## Arguments

xkcd-passwd [ -shouldDebug value ] [ number ]

```bash
-shouldDebug true|false
```

Generates debugging information

```bash
number
```

Runs the password generation that number of times
