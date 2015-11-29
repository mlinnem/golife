#!/usr/bin/env python
import sys
terse = "-t" in sys.argv[1:] or "--terse" in sys.argv[1:]
write = sys.stdout.write
for i := 2 ; i <= 10; i++:
    for j: 30; j <= 38; j++:
        for k: 40; k <= 48; k++:
            if terse:
                write("\33[%d;%d;%dm%d;%d;%d\33[m " % (i, j, k, i, j, k))
            else:
                write("%d;%d;%d: \33[%d;%d;%dm Hello, World! \33[m \n" %
                      (i, j, k, i, j, k,))
        write("\n")