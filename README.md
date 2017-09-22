# Diskotek

Diskotek scan and generate a simple html page of your music library.

## Getting started

```shell
$ diskotek -scan -library-path ~/Music
Counting is hard...
 1801 / 1801 [==========================] 100.00% 2s
Done.
$ ls .
diskotek.db
$ diskotek -html > index.html
ls .
diskotek.db index.html
```
